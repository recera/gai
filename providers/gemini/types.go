package gemini

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/recera/gai/core"
)

// GenerateContentRequest represents a request to the Gemini generateContent API.
type GenerateContentRequest struct {
	Contents          []Content          `json:"contents"`
	Tools             []Tool             `json:"tools,omitempty"`
	ToolConfig        *ToolConfig        `json:"toolConfig,omitempty"`
	SafetySettings    []SafetySetting    `json:"safetySettings,omitempty"`
	GenerationConfig  *GenerationConfig  `json:"generationConfig,omitempty"`
	SystemInstruction *Content           `json:"systemInstruction,omitempty"`
}

// Content represents a message in Gemini's format.
type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

// Part represents a content part in Gemini's format.
type Part struct {
	Text           string          `json:"text,omitempty"`
	InlineData     *InlineData     `json:"inlineData,omitempty"`
	FileData       *FileData       `json:"fileData,omitempty"`
	FunctionCall   *FunctionCall   `json:"functionCall,omitempty"`
	FunctionResult *FunctionResult `json:"functionResponse,omitempty"`
}

// InlineData represents inline binary data.
type InlineData struct {
	MIMEType string `json:"mimeType"`
	Data     string `json:"data"` // Base64 encoded
}

// FileData represents a reference to an uploaded file.
type FileData struct {
	MIMEType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

// Tool represents a tool/function definition.
type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations"`
}

// FunctionDeclaration defines a function that can be called.
type FunctionDeclaration struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}

// FunctionCall represents a function call request.
type FunctionCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// FunctionResult represents a function execution result.
type FunctionResult struct {
	Name     string          `json:"name"`
	Response json.RawMessage `json:"response"`
}

// ToolConfig specifies how tools should be used.
type ToolConfig struct {
	FunctionCallingConfig *FunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

// FunctionCallingConfig controls function calling behavior.
type FunctionCallingConfig struct {
	Mode string `json:"mode"` // AUTO, ANY, NONE
}

// SafetySetting configures safety thresholds.
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GenerationConfig controls generation parameters.
type GenerationConfig struct {
	Temperature      *float32        `json:"temperature,omitempty"`
	TopP             *float32        `json:"topP,omitempty"`
	TopK             *int32          `json:"topK,omitempty"`
	CandidateCount   *int32          `json:"candidateCount,omitempty"`
	MaxOutputTokens  *int32          `json:"maxOutputTokens,omitempty"`
	StopSequences    []string        `json:"stopSequences,omitempty"`
	ResponseMIMEType string          `json:"responseMimeType,omitempty"`
	ResponseSchema   json.RawMessage `json:"responseSchema,omitempty"`
}

// GenerateContentResponse represents the API response.
type GenerateContentResponse struct {
	Candidates     []Candidate     `json:"candidates"`
	PromptFeedback *PromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *UsageMetadata  `json:"usageMetadata,omitempty"`
}

// Candidate represents a generation candidate.
type Candidate struct {
	Content        Content         `json:"content"`
	FinishReason   string          `json:"finishReason"`
	Index          int             `json:"index"`
	SafetyRatings  []SafetyRating  `json:"safetyRatings"`
	CitationMetadata *CitationMetadata `json:"citationMetadata,omitempty"`
	TokenCount     int             `json:"tokenCount"`
}

// SafetyRating represents a safety assessment.
type SafetyRating struct {
	Category    string  `json:"category"`
	Probability string  `json:"probability"`
	Blocked     bool    `json:"blocked"`
	Score       float32 `json:"score,omitempty"`
}

// CitationMetadata contains citation information.
type CitationMetadata struct {
	CitationSources []CitationSource `json:"citationSources"`
}

// CitationSource represents a single citation.
type CitationSource struct {
	StartIndex int    `json:"startIndex"`
	EndIndex   int    `json:"endIndex"`
	URI        string `json:"uri"`
	License    string `json:"license,omitempty"`
	Title      string `json:"title,omitempty"`
}

// PromptFeedback provides feedback on the prompt.
type PromptFeedback struct {
	BlockReason   string         `json:"blockReason,omitempty"`
	SafetyRatings []SafetyRating `json:"safetyRatings"`
}

// UsageMetadata contains token usage information.
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// StreamingResponse represents a streaming response chunk.
type StreamingResponse struct {
	Candidates     []Candidate     `json:"candidates"`
	UsageMetadata  *UsageMetadata  `json:"usageMetadata,omitempty"`
	PromptFeedback *PromptFeedback `json:"promptFeedback,omitempty"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		Details []struct {
			Type     string `json:"@type"`
			Reason   string `json:"reason"`
			Domain   string `json:"domain"`
			Metadata map[string]string `json:"metadata"`
		} `json:"details"`
	} `json:"error"`
}

// FileInfo stores information about an uploaded file.
type FileInfo struct {
	ID        string
	URI       string
	MIMEType  string
	Size      int64
	ExpiresAt time.Time
}

// FileStore manages uploaded files.
type FileStore struct {
	files map[string]*FileInfo
	mu    sync.RWMutex
}

// NewFileStore creates a new file store.
func NewFileStore() *FileStore {
	return &FileStore{
		files: make(map[string]*FileInfo),
	}
}

// Store saves file information.
func (fs *FileStore) Store(id string, info *FileInfo) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.files[id] = info
}

// Get retrieves file information.
func (fs *FileStore) Get(id string) (*FileInfo, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	info, ok := fs.files[id]
	return info, ok
}

// Clean removes expired files.
func (fs *FileStore) Clean() {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	now := time.Now()
	for id, info := range fs.files {
		if now.After(info.ExpiresAt) {
			delete(fs.files, id)
		}
	}
}

// convertRole maps GAI roles to Gemini roles.
func convertRole(role core.Role) string {
	switch role {
	case core.System:
		return "model" // Gemini doesn't have system role in messages
	case core.User:
		return "user"
	case core.Assistant:
		return "model"
	case core.Tool:
		return "function"
	default:
		return "user"
	}
}

// convertSafetyLevel maps GAI safety levels to Gemini thresholds.
func convertSafetyLevel(level core.SafetyLevel) string {
	switch level {
	case core.SafetyBlockNone:
		return "BLOCK_NONE"
	case core.SafetyBlockFew:
		return "BLOCK_ONLY_HIGH"
	case core.SafetyBlockSome:
		return "BLOCK_MEDIUM_AND_ABOVE"
	case core.SafetyBlockMost:
		return "BLOCK_LOW_AND_ABOVE"
	case core.SafetyBlockAlways:
		return "BLOCK_LOW_AND_ABOVE" // Gemini doesn't have "always block"
	default:
		return "BLOCK_MEDIUM_AND_ABOVE"
	}
}

// convertSafetyCategory maps category names.
func convertSafetyCategory(category string) string {
	switch strings.ToLower(category) {
	case "harassment":
		return "HARM_CATEGORY_HARASSMENT"
	case "hate":
		return "HARM_CATEGORY_HATE_SPEECH"
	case "sexual":
		return "HARM_CATEGORY_SEXUALLY_EXPLICIT"
	case "dangerous":
		return "HARM_CATEGORY_DANGEROUS_CONTENT"
	default:
		return "HARM_CATEGORY_UNSPECIFIED"
	}
}

// convertToolChoice maps GAI tool choice to Gemini mode.
func convertToolChoice(choice core.ToolChoice) string {
	switch choice {
	case core.ToolAuto:
		return "AUTO"
	case core.ToolNone:
		return "NONE"
	case core.ToolRequired:
		return "ANY"
	default:
		return "AUTO"
	}
}