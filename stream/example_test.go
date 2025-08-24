// Package stream provides streaming utilities for AI responses.
// This file contains examples for using SSE and NDJSON streaming.
package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/recera/gai/core"
)

// Example_SSE demonstrates how to use SSE streaming with an HTTP server.
func Example_sSE() {
	// Create a mock provider (in production, use a real AI provider)
	provider := &exampleProvider{}
	
	// Create SSE handler
	handler := SSEHandler(provider, func(r *http.Request) (core.Request, error) {
		// Extract question from query parameter
		question := r.URL.Query().Get("q")
		if question == "" {
			question = "Hello, how can I help you?"
		}
		
		// Create AI request
		return core.Request{
			Messages: []core.Message{
				{
					Role: core.System,
					Parts: []core.Part{
						core.Text{Text: "You are a helpful assistant."},
					},
				},
				{
					Role: core.User,
					Parts: []core.Part{
						core.Text{Text: question},
					},
				},
			},
			Stream:      true,
			Temperature: 0.7,
			MaxTokens:   500,
		}, nil
	})
	
	// Set up HTTP server
	http.HandleFunc("/chat", handler)
	
	// The following HTML/JavaScript would be used in the browser:
	htmlExample := `
<!DOCTYPE html>
<html>
<head>
    <title>SSE Chat Example</title>
</head>
<body>
    <div id="messages"></div>
    <input type="text" id="question" placeholder="Ask a question...">
    <button onclick="askQuestion()">Send</button>
    
    <script>
    function askQuestion() {
        const question = document.getElementById('question').value;
        const messagesDiv = document.getElementById('messages');
        
        // Clear previous response
        messagesDiv.innerHTML = '';
        
        // Create EventSource
        const eventSource = new EventSource('/chat?q=' + encodeURIComponent(question));
        
        eventSource.addEventListener('EventTextDelta', function(e) {
            const data = JSON.parse(e.data);
            messagesDiv.innerHTML += data.data.text;
        });
        
        eventSource.addEventListener('EventFinish', function(e) {
            const data = JSON.parse(e.data);
            console.log('Usage:', data.data.usage);
        });
        
        eventSource.addEventListener('done', function(e) {
            eventSource.close();
        });
        
        eventSource.addEventListener('error', function(e) {
            console.error('SSE Error:', e);
            eventSource.close();
        });
    }
    </script>
</body>
</html>
`
	
	fmt.Println("SSE Server Example:")
	fmt.Println("1. Start server with: http.ListenAndServe(\":8080\", nil)")
	fmt.Println("2. Open browser to http://localhost:8080")
	fmt.Println("3. Use this HTML:")
	fmt.Println(htmlExample)
	
	// Output:
	// SSE Server Example:
	// 1. Start server with: http.ListenAndServe(":8080", nil)
	// 2. Open browser to http://localhost:8080
	// 3. Use this HTML:
	// <!DOCTYPE html>
	// <html>
	// <head>
	//     <title>SSE Chat Example</title>
	// </head>
	// <body>
	//     <div id="messages"></div>
	//     <input type="text" id="question" placeholder="Ask a question...">
	//     <button onclick="askQuestion()">Send</button>
	//     
	//     <script>
	//     function askQuestion() {
	//         const question = document.getElementById('question').value;
	//         const messagesDiv = document.getElementById('messages');
	//         
	//         // Clear previous response
	//         messagesDiv.innerHTML = '';
	//         
	//         // Create EventSource
	//         const eventSource = new EventSource('/chat?q=' + encodeURIComponent(question));
	//         
	//         eventSource.addEventListener('EventTextDelta', function(e) {
	//             const data = JSON.parse(e.data);
	//             messagesDiv.innerHTML += data.data.text;
	//         });
	//         
	//         eventSource.addEventListener('EventFinish', function(e) {
	//             const data = JSON.parse(e.data);
	//             console.log('Usage:', data.data.usage);
	//         });
	//         
	//         eventSource.addEventListener('done', function(e) {
	//             eventSource.close();
	//         });
	//         
	//         eventSource.addEventListener('error', function(e) {
	//             console.error('SSE Error:', e);
	//             eventSource.close();
	//         });
	//     }
	//     </script>
	// </body>
	// </html>
}

// Example_NDJSON demonstrates how to use NDJSON streaming with an HTTP server.
func Example_nDJSON() {
	// Create a mock provider
	provider := &exampleProvider{}
	
	// Create NDJSON handler
	handler := NDJSONHandler(provider, func(r *http.Request) (core.Request, error) {
		// Parse JSON request body
		var reqBody struct {
			Messages []core.Message `json:"messages"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			return core.Request{}, err
		}
		
		return core.Request{
			Messages: reqBody.Messages,
			Stream:   true,
		}, nil
	})
	
	// Set up HTTP server
	http.HandleFunc("/api/chat", handler)
	
	// JavaScript fetch example for browser:
	jsExample := `
async function streamChat(messages) {
    const response = await fetch('/api/chat', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ messages })
    });
    
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    
    while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        
        // Process complete lines
        for (let i = 0; i < lines.length - 1; i++) {
            const line = lines[i].trim();
            if (!line) continue;
            
            try {
                const event = JSON.parse(line);
                
                switch(event.type) {
                    case 'EventTextDelta':
                        document.getElementById('output').innerHTML += event.text;
                        break;
                    case 'EventToolCall':
                        console.log('Tool call:', event.tool_call);
                        break;
                    case 'EventFinish':
                        console.log('Finished. Usage:', event.usage);
                        break;
                    case 'done':
                        console.log('Stream complete');
                        return;
                }
            } catch (e) {
                console.error('Failed to parse line:', line, e);
            }
        }
        
        // Keep incomplete line in buffer
        buffer = lines[lines.length - 1];
    }
}

// Usage:
streamChat([
    { role: 'user', parts: [{ text: 'Hello!' }] }
]);
`
	
	fmt.Println("NDJSON Server Example:")
	fmt.Println("1. Start server with: http.ListenAndServe(\":8080\", nil)")
	fmt.Println("2. Use this JavaScript in browser:")
	fmt.Println(jsExample)
	
	// Output:
	// NDJSON Server Example:
	// 1. Start server with: http.ListenAndServe(":8080", nil)
	// 2. Use this JavaScript in browser:
	// async function streamChat(messages) {
	//     const response = await fetch('/api/chat', {
	//         method: 'POST',
	//         headers: {
	//             'Content-Type': 'application/json',
	//         },
	//         body: JSON.stringify({ messages })
	//     });
	//     
	//     const reader = response.body.getReader();
	//     const decoder = new TextDecoder();
	//     let buffer = '';
	//     
	//     while (true) {
	//         const { done, value } = await reader.read();
	//         if (done) break;
	//         
	//         buffer += decoder.decode(value, { stream: true });
	//         const lines = buffer.split('\n');
	//         
	//         // Process complete lines
	//         for (let i = 0; i < lines.length - 1; i++) {
	//             const line = lines[i].trim();
	//             if (!line) continue;
	//             
	//             try {
	//                 const event = JSON.parse(line);
	//                 
	//                 switch(event.type) {
	//                     case 'EventTextDelta':
	//                         document.getElementById('output').innerHTML += event.text;
	//                         break;
	//                     case 'EventToolCall':
	//                         console.log('Tool call:', event.tool_call);
	//                         break;
	//                     case 'EventFinish':
	//                         console.log('Finished. Usage:', event.usage);
	//                         break;
	//                     case 'done':
	//                         console.log('Stream complete');
	//                         return;
	//                 }
	//             } catch (e) {
	//                 console.error('Failed to parse line:', line, e);
	//             }
	//         }
	//         
	//         // Keep incomplete line in buffer
	//         buffer = lines[lines.length - 1];
	//     }
	// }
	// 
	// // Usage:
	// streamChat([
	//     { role: 'user', parts: [{ text: 'Hello!' }] }
	// ]);
}

// Example_customSSEOptions demonstrates using custom SSE options.
func Example_customSSEOptions() {
	// Create a text stream
	stream := &exampleStream{}
	
	// Configure custom SSE options
	options := SSEOptions{
		HeartbeatInterval: 30 * time.Second,  // Longer heartbeat interval
		FlushAfterWrite:   true,               // Always flush
		MaxRetries:        5,                  // More retries
		BufferSize:        8192,               // Larger buffer
		IncludeID:         true,               // Include event IDs for replay
	}
	
	// Use in HTTP handler
	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		if err := SSE(w, stream, options); err != nil {
			log.Printf("SSE error: %v", err)
		}
	})
	
	fmt.Println("SSE with custom options configured")
	// Output: SSE with custom options configured
}

// Example_customNDJSONOptions demonstrates using custom NDJSON options.
func Example_customNDJSONOptions() {
	// Create a text stream
	stream := &exampleStream{}
	
	// Configure custom NDJSON options
	options := NDJSONOptions{
		BufferSize:       16384,               // Larger buffer for performance
		FlushInterval:    50 * time.Millisecond, // Fast flushing
		CompactJSON:      false,               // Pretty print for debugging
		IncludeTimestamp: true,                // Add timestamps to each event
	}
	
	// Use in HTTP handler
	http.HandleFunc("/api/stream", func(w http.ResponseWriter, r *http.Request) {
		if err := NDJSON(w, stream, options); err != nil {
			log.Printf("NDJSON error: %v", err)
		}
	})
	
	fmt.Println("NDJSON with custom options configured")
	// Output: NDJSON with custom options configured
}

// exampleProvider is a mock provider for examples.
type exampleProvider struct{}

func (p *exampleProvider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	return &core.TextResult{Text: "Example response"}, nil
}

func (p *exampleProvider) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
	stream := &exampleStream{}
	
	// Simulate streaming response
	go func() {
		defer stream.close()
		
		responses := []string{"Hello", " there", "! How", " can", " I", " help", " you", " today", "?"}
		for _, text := range responses {
			stream.send(core.Event{
				Type:      core.EventTextDelta,
				TextDelta: text,
				Timestamp: time.Now(),
			})
			time.Sleep(50 * time.Millisecond) // Simulate processing
		}
		
		stream.send(core.Event{
			Type: core.EventFinish,
			Usage: &core.Usage{
				InputTokens:  10,
				OutputTokens: 9,
				TotalTokens:  19,
			},
			Timestamp: time.Now(),
		})
	}()
	
	return stream, nil
}

func (p *exampleProvider) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *exampleProvider) StreamObject(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
	return nil, fmt.Errorf("not implemented")
}

// exampleStream is a simple text stream for examples.
type exampleStream struct {
	events chan core.Event
	closed bool
}

func (s *exampleStream) Events() <-chan core.Event {
	if s.events == nil {
		s.events = make(chan core.Event, 100)
	}
	return s.events
}

func (s *exampleStream) Close() error {
	s.close()
	return nil
}

func (s *exampleStream) send(event core.Event) {
	if s.events == nil {
		s.events = make(chan core.Event, 100)
	}
	if !s.closed {
		s.events <- event
	}
}

func (s *exampleStream) close() {
	if !s.closed {
		close(s.events)
		s.closed = true
	}
}