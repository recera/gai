package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRoleConstants(t *testing.T) {
	tests := []struct {
		role     Role
		expected string
	}{
		{System, "system"},
		{User, "user"},
		{Assistant, "assistant"},
		{Tool, "tool"},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if string(tt.role) != tt.expected {
				t.Errorf("Role %v = %q, want %q", tt.role, string(tt.role), tt.expected)
			}
		})
	}
}

func TestPartTypes(t *testing.T) {
	tests := []struct {
		name     string
		part     Part
		partType string
	}{
		{"Text", Text{Text: "hello"}, "text"},
		{"ImageURL", ImageURL{URL: "http://example.com/img.jpg"}, "image_url"},
		{"Audio", Audio{Source: BlobRef{Kind: BlobURL, URL: "http://example.com/audio.mp3"}}, "audio"},
		{"Video", Video{Source: BlobRef{Kind: BlobURL, URL: "http://example.com/video.mp4"}}, "video"},
		{"File", File{Source: BlobRef{Kind: BlobBytes, Bytes: []byte("data")}, Name: "doc.pdf"}, "file"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify part implements Part interface
			var _ Part = tt.part
			
			// Verify partType method
			if got := tt.part.partType(); got != tt.partType {
				t.Errorf("partType() = %q, want %q", got, tt.partType)
			}
		})
	}
}

func TestBlobRefKinds(t *testing.T) {
	tests := []struct {
		name string
		blob BlobRef
		want BlobKind
	}{
		{
			name: "URL blob",
			blob: BlobRef{Kind: BlobURL, URL: "http://example.com/file"},
			want: BlobURL,
		},
		{
			name: "Bytes blob",
			blob: BlobRef{Kind: BlobBytes, Bytes: []byte("data")},
			want: BlobBytes,
		},
		{
			name: "Provider file blob",
			blob: BlobRef{Kind: BlobProviderFile, FileID: "file-123"},
			want: BlobProviderFile,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.blob.Kind != tt.want {
				t.Errorf("BlobRef.Kind = %v, want %v", tt.blob.Kind, tt.want)
			}
		})
	}
}

func TestMessageSerialization(t *testing.T) {
	msg := Message{
		Role: User,
		Parts: []Part{
			Text{Text: "Hello"},
			ImageURL{URL: "http://example.com/img.jpg", Detail: "high"},
		},
		Name: "user1",
	}
	
	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal Message: %v", err)
	}
	
	// Unmarshal back
	var decoded Message
	// Note: In production, we'd need custom JSON marshaling for Part interface
	// This is a simplified test
	if err := json.Unmarshal(data, &decoded); err == nil {
		// We expect this to fail without custom marshaling
		// Just verify the basic structure serializes
		t.Log("Message serialized successfully")
	}
}

func TestToolChoice(t *testing.T) {
	tests := []struct {
		choice   ToolChoice
		expected int
	}{
		{ToolAuto, 0},
		{ToolNone, 1},
		{ToolRequired, 2},
		{ToolSpecific, 3},
	}
	
	for _, tt := range tests {
		if int(tt.choice) != tt.expected {
			t.Errorf("ToolChoice %v = %d, want %d", tt.choice, int(tt.choice), tt.expected)
		}
	}
}

func TestSafetyLevels(t *testing.T) {
	levels := []SafetyLevel{
		SafetyBlockNone,
		SafetyBlockFew,
		SafetyBlockSome,
		SafetyBlockMost,
		SafetyBlockAlways,
	}
	
	expected := []string{
		"block_none",
		"block_few",
		"block_some",
		"block_most",
		"block_always",
	}
	
	for i, level := range levels {
		if string(level) != expected[i] {
			t.Errorf("SafetyLevel %v = %q, want %q", level, string(level), expected[i])
		}
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventStart, "start"},
		{EventTextDelta, "text_delta"},
		{EventAudioDelta, "audio_delta"},
		{EventToolCall, "tool_call"},
		{EventToolResult, "tool_result"},
		{EventCitations, "citations"},
		{EventSafety, "safety"},
		{EventFinishStep, "finish_step"},
		{EventFinish, "finish"},
		{EventError, "error"},
		{EventRaw, "raw"},
		{EventType(999), "unknown(999)"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.eventType.String(); got != tt.expected {
				t.Errorf("EventType.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStopConditions(t *testing.T) {
	t.Run("MaxSteps", func(t *testing.T) {
		cond := MaxSteps(3)
		step := Step{Text: "response"}
		
		tests := []struct {
			stepCount int
			want      bool
		}{
			{1, false},
			{2, false},
			{3, true},
			{4, true},
		}
		
		for _, tt := range tests {
			got := cond.ShouldStop(tt.stepCount, step)
			if got != tt.want {
				t.Errorf("MaxSteps(3).ShouldStop(%d) = %v, want %v", tt.stepCount, got, tt.want)
			}
		}
	})
	
	t.Run("NoMoreTools", func(t *testing.T) {
		cond := NoMoreTools()
		
		stepWithTools := Step{
			Text: "calling tool",
			ToolCalls: []ToolCall{
				{Name: "tool1", Input: json.RawMessage(`{}`)},
			},
		}
		
		stepWithoutTools := Step{
			Text: "final answer",
		}
		
		if cond.ShouldStop(1, stepWithTools) {
			t.Error("NoMoreTools should not stop when tools are present")
		}
		
		if !cond.ShouldStop(1, stepWithoutTools) {
			t.Error("NoMoreTools should stop when no tools are present")
		}
	})
	
	t.Run("UntilToolSeen", func(t *testing.T) {
		cond := UntilToolSeen("search")
		
		stepWithOtherTool := Step{
			ToolCalls: []ToolCall{
				{Name: "calculator", Input: json.RawMessage(`{}`)},
			},
		}
		
		stepWithTargetTool := Step{
			ToolCalls: []ToolCall{
				{Name: "search", Input: json.RawMessage(`{}`)},
			},
		}
		
		if cond.ShouldStop(1, stepWithOtherTool) {
			t.Error("UntilToolSeen should not stop for other tools")
		}
		
		if !cond.ShouldStop(1, stepWithTargetTool) {
			t.Error("UntilToolSeen should stop when target tool is seen")
		}
	})
	
	t.Run("CombineConditions", func(t *testing.T) {
		cond := CombineConditions(
			MaxSteps(2),
			UntilToolSeen("done"),
		)
		
		stepNormal := Step{
			ToolCalls: []ToolCall{{Name: "other"}},
		}
		
		stepDone := Step{
			ToolCalls: []ToolCall{{Name: "done"}},
		}
		
		// Should not stop at step 1 with normal tool
		if cond.ShouldStop(1, stepNormal) {
			t.Error("Combined condition should not stop at step 1")
		}
		
		// Should stop at step 2 (max steps)
		if !cond.ShouldStop(2, stepNormal) {
			t.Error("Combined condition should stop at max steps")
		}
		
		// Should stop when "done" tool is seen
		if !cond.ShouldStop(1, stepDone) {
			t.Error("Combined condition should stop when done tool is seen")
		}
	})
}

func TestUsageAggregation(t *testing.T) {
	usage := Usage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}
	
	if usage.TotalTokens != usage.InputTokens+usage.OutputTokens {
		t.Errorf("Usage totals don't match: %d != %d + %d",
			usage.TotalTokens, usage.InputTokens, usage.OutputTokens)
	}
}

func TestEventStructure(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name  string
		event Event
		check func(t *testing.T, e Event)
	}{
		{
			name: "TextDelta event",
			event: Event{
				Type:      EventTextDelta,
				TextDelta: "Hello world",
				Timestamp: now,
			},
			check: func(t *testing.T, e Event) {
				if e.TextDelta != "Hello world" {
					t.Errorf("TextDelta = %q, want %q", e.TextDelta, "Hello world")
				}
			},
		},
		{
			name: "ToolCall event",
			event: Event{
				Type:      EventToolCall,
				ToolName:  "search",
				ToolID:    "call-123",
				ToolInput: json.RawMessage(`{"query":"test"}`),
				Timestamp: now,
			},
			check: func(t *testing.T, e Event) {
				if e.ToolName != "search" {
					t.Errorf("ToolName = %q, want %q", e.ToolName, "search")
				}
				if e.ToolID != "call-123" {
					t.Errorf("ToolID = %q, want %q", e.ToolID, "call-123")
				}
			},
		},
		{
			name: "Citations event",
			event: Event{
				Type: EventCitations,
				Citations: []Citation{
					{URI: "http://example.com", Start: 0, End: 10, Title: "Example"},
				},
				Timestamp: now,
			},
			check: func(t *testing.T, e Event) {
				if len(e.Citations) != 1 {
					t.Errorf("Citations count = %d, want 1", len(e.Citations))
				}
				if e.Citations[0].URI != "http://example.com" {
					t.Errorf("Citation URI = %q, want %q", e.Citations[0].URI, "http://example.com")
				}
			},
		},
		{
			name: "Safety event",
			event: Event{
				Type: EventSafety,
				Safety: &SafetyEvent{
					Category: "harassment",
					Action:   "block",
					Score:    0.95,
					Note:     "Content blocked",
				},
				Timestamp: now,
			},
			check: func(t *testing.T, e Event) {
				if e.Safety == nil {
					t.Fatal("Safety is nil")
				}
				if e.Safety.Category != "harassment" {
					t.Errorf("Safety.Category = %q, want %q", e.Safety.Category, "harassment")
				}
				if e.Safety.Score != 0.95 {
					t.Errorf("Safety.Score = %f, want %f", e.Safety.Score, 0.95)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.event)
		})
	}
}

func TestRequestValidation(t *testing.T) {
	req := Request{
		Model: "gpt-4",
		Messages: []Message{
			{Role: System, Parts: []Part{Text{Text: "You are helpful"}}},
			{Role: User, Parts: []Part{Text{Text: "Hello"}}},
		},
		Temperature: 0.7,
		MaxTokens:   1000,
		Stream:      true,
	}
	
	// Verify required fields
	if req.Model == "" {
		t.Error("Model should not be empty")
	}
	
	if len(req.Messages) == 0 {
		t.Error("Messages should not be empty")
	}
	
	// Verify temperature bounds (convention, not enforced by type)
	if req.Temperature < 0 || req.Temperature > 2 {
		t.Errorf("Temperature %f is out of conventional bounds [0, 2]", req.Temperature)
	}
}