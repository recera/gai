package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/recera/gai/core"
)

// geminiStream implements core.TextStream for Gemini streaming responses.
type geminiStream struct {
	events   chan core.Event
	cancel   context.CancelFunc
	response *http.Response
	done     chan struct{}
}

// createStream creates a streaming response.
func (p *Provider) createStream(ctx context.Context, req core.Request) (core.TextStream, error) {
	// Convert request to Gemini format
	geminiReq := p.convertRequest(req)

	// Marshal request
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, err
	}

	// Create stream context
	streamCtx, cancel := context.WithCancel(ctx)

	// Make streaming request
	stream, err := p.doStreamRequest(streamCtx, body)
	if err != nil {
		cancel()
		return nil, err
	}

	// Start processing events
	go stream.processEvents()

	return stream, nil
}

// doStreamRequest performs the streaming HTTP request.
func (p *Provider) doStreamRequest(ctx context.Context, body []byte) (*geminiStream, error) {
	model := p.model
	if model == "" {
		model = "gemini-1.5-flash"
	}

	// Use streamGenerateContent endpoint with alt=sse for SSE streaming
	url := fmt.Sprintf("%s/%s/models/%s:streamGenerateContent?alt=sse&key=%s",
		p.baseURL, apiVersion, model, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
		}
		return nil, mapError(&errResp, resp.StatusCode)
	}

	// Create stream
	_, cancel := context.WithCancel(ctx)
	stream := &geminiStream{
		events:   make(chan core.Event, 100),
		cancel:   cancel,
		response: resp,
		done:     make(chan struct{}),
	}

	return stream, nil
}

// Events returns the event channel.
func (s *geminiStream) Events() <-chan core.Event {
	return s.events
}

// Close terminates the stream.
func (s *geminiStream) Close() error {
	s.cancel()
	<-s.done
	return s.response.Body.Close()
}

// processEvents reads and processes SSE events from the response.
func (s *geminiStream) processEvents() {
	defer close(s.done)
	defer close(s.events)
	defer s.response.Body.Close()

	scanner := bufio.NewScanner(s.response.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max

	var eventData strings.Builder
	totalUsage := core.Usage{}
	allCitations := []core.Citation{}
	
	// Send start event
	s.events <- core.Event{
		Type:      core.EventStart,
		Timestamp: time.Now(),
	}

	for scanner.Scan() {
		line := scanner.Text()

		// SSE format: "data: {...}"
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			
			// Skip empty data
			if data == "" || data == "[DONE]" {
				continue
			}

			// Parse streaming response
			var chunk StreamingResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				s.events <- core.Event{
					Type:      core.EventError,
					Err:       fmt.Errorf("failed to parse chunk: %w", err),
					Timestamp: time.Now(),
				}
				continue
			}

			// Process prompt feedback (safety blocking)
			if chunk.PromptFeedback != nil && chunk.PromptFeedback.BlockReason != "" {
				s.events <- core.Event{
					Type: core.EventSafety,
					Safety: &core.SafetyEvent{
						Category: "prompt",
						Action:   "block",
						Note:     chunk.PromptFeedback.BlockReason,
					},
					Timestamp: time.Now(),
				}
				continue
			}

			// Process candidates
			for _, candidate := range chunk.Candidates {
				// Emit safety ratings as events
				for _, rating := range candidate.SafetyRatings {
					if rating.Blocked {
						s.events <- core.Event{
							Type: core.EventSafety,
							Safety: &core.SafetyEvent{
								Category: convertSafetyCategory(rating.Category),
								Action:   "block",
								Score:    rating.Score,
							},
							Timestamp: time.Now(),
						}
					}
				}

				// Process content parts
				for _, part := range candidate.Content.Parts {
					// Text content
					if part.Text != "" {
						s.events <- core.Event{
							Type:      core.EventTextDelta,
							TextDelta: part.Text,
							Timestamp: time.Now(),
						}
						eventData.WriteString(part.Text)
					}

					// Function calls
					if part.FunctionCall != nil {
						s.events <- core.Event{
							Type:      core.EventToolCall,
							ToolName:  part.FunctionCall.Name,
							ToolInput: part.FunctionCall.Args,
							Timestamp: time.Now(),
						}
					}

					// Function results
					if part.FunctionResult != nil {
						s.events <- core.Event{
							Type:       core.EventToolResult,
							ToolName:   part.FunctionResult.Name,
							ToolResult: part.FunctionResult.Response,
							Timestamp:  time.Now(),
						}
					}
				}

				// Process citations
				if candidate.CitationMetadata != nil {
					citations := convertCitations(candidate.CitationMetadata, eventData.String())
					if len(citations) > 0 {
						allCitations = append(allCitations, citations...)
						s.events <- core.Event{
							Type:      core.EventCitations,
							Citations: citations,
							Timestamp: time.Now(),
						}
					}
				}
			}

			// Update usage
			if chunk.UsageMetadata != nil {
				totalUsage.InputTokens = chunk.UsageMetadata.PromptTokenCount
				totalUsage.OutputTokens = chunk.UsageMetadata.CandidatesTokenCount
				totalUsage.TotalTokens = chunk.UsageMetadata.TotalTokenCount
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		s.events <- core.Event{
			Type:      core.EventError,
			Err:       err,
			Timestamp: time.Now(),
		}
		return
	}

	// Send finish event with usage
	s.events <- core.Event{
		Type:      core.EventFinish,
		Usage:     &totalUsage,
		Timestamp: time.Now(),
	}
}

// convertCitations converts Gemini citations to GAI format.
func convertCitations(metadata *CitationMetadata, text string) []core.Citation {
	if metadata == nil || len(metadata.CitationSources) == 0 {
		return nil
	}

	citations := make([]core.Citation, 0, len(metadata.CitationSources))
	for _, source := range metadata.CitationSources {
		citations = append(citations, core.Citation{
			URI:   source.URI,
			Start: source.StartIndex,
			End:   source.EndIndex,
			Title: source.Title,
		})
	}
	return citations
}

