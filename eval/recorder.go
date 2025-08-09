package eval

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

// Recorder middleware writes interactions to an NDJSON log for eval datasets.
type Recorder struct {
	mu   sync.Mutex
	file *os.File
}

func NewRecorder(path string) (*Recorder, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Recorder{file: f}, nil
}

func (r *Recorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

type record struct {
	Timestamp    time.Time       `json:"ts"`
	Provider     string          `json:"provider"`
	Model        string          `json:"model"`
	SessionID    string          `json:"session_id,omitempty"`
	Metadata     map[string]any  `json:"metadata,omitempty"`
	Messages     []core.Message  `json:"messages"`
	Settings     map[string]any  `json:"settings,omitempty"`
	Response     string          `json:"response"`
	Finish       string          `json:"finish_reason"`
	Usage        core.TokenUsage `json:"usage"`
	ExpectedTxt  string          `json:"expected_text,omitempty"`
	ExpectedJSON any             `json:"expected_json,omitempty"`
	Error        string          `json:"error,omitempty"`
}

// Wrap returns a ProviderClient that records blocking calls to the NDJSON file.
func (r *Recorder) Wrap(next core.ProviderClient) core.ProviderClient {
	return &recProv{next: next, r: r}
}

type recProv struct {
	next core.ProviderClient
	r    *Recorder
}

func (p *recProv) GetCompletion(ctx context.Context, parts core.LLMCallParts) (core.LLMResponse, error) {
	resp, err := p.next.GetCompletion(ctx, parts)
	rec := record{
		Timestamp:    time.Now(),
		Provider:     parts.Provider,
		Model:        parts.Model,
		SessionID:    parts.SessionID,
		Metadata:     parts.Metadata,
		Messages:     parts.Messages,
		Settings:     map[string]any{"max_tokens": parts.MaxTokens, "temperature": parts.Temperature},
		Response:     resp.Content,
		Finish:       resp.FinishReason,
		Usage:        resp.Usage,
		ExpectedTxt:  parts.ExpectedText,
		ExpectedJSON: parts.ExpectedJSON,
	}
	if err != nil {
		rec.Error = err.Error()
	}
	// write
	b, _ := json.Marshal(rec)
	p.r.mu.Lock()
	p.r.file.Write(b)
	p.r.file.Write([]byte("\n"))
	p.r.mu.Unlock()
	return resp, err
}

func (p *recProv) StreamCompletion(ctx context.Context, parts core.LLMCallParts, handler core.StreamHandler) error {
	// For now, forward to underlying stream and do not record stream deltas line-by-line.
	// Optional: buffer content and write one record at end.
	var buf string
	var finish string
	var usage core.TokenUsage
	err := p.next.StreamCompletion(ctx, parts, func(ch core.StreamChunk) error {
		if ch.Type == "content" {
			buf += ch.Delta
		}
		if ch.Type == "end" {
			finish = ch.FinishReason
		}
		return handler(ch)
	})
	rec := record{
		Timestamp: time.Now(), Provider: parts.Provider, Model: parts.Model,
		SessionID: parts.SessionID, Metadata: parts.Metadata, Messages: parts.Messages,
		Settings: map[string]any{"max_tokens": parts.MaxTokens, "temperature": parts.Temperature},
		Response: buf, Finish: finish, Usage: usage, ExpectedTxt: parts.ExpectedText, ExpectedJSON: parts.ExpectedJSON,
	}
	if err != nil {
		rec.Error = err.Error()
	}
	b, _ := json.Marshal(rec)
	p.r.mu.Lock()
	p.r.file.Write(b)
	p.r.file.Write([]byte("\n"))
	p.r.mu.Unlock()
	return err
}
