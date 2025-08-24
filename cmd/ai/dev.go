package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/middleware"
	"github.com/recera/gai/providers/openai"
	"github.com/recera/gai/stream"
	"github.com/spf13/cobra"
)

// devCmd represents the dev command group
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Development tools and utilities",
	Long:  `Development tools for testing and debugging GAI applications.`,
}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a development server with SSE/NDJSON endpoints",
	Long: `Starts a development server that provides SSE (Server-Sent Events) and NDJSON
streaming endpoints for testing AI interactions.

The server provides:
  - /api/chat - SSE endpoint for streaming chat responses
  - /api/chat/ndjson - NDJSON endpoint for streaming chat responses
  - /api/generate - Non-streaming text generation endpoint
  - / - Web interface for testing

Environment variables:
  OPENAI_API_KEY - Required for OpenAI provider
  PORT - Server port (default: 8080)`,
	RunE: runServe,
}

var (
	port     string
	provider string
	model    string
)

func init() {
	rootCmd.AddCommand(devCmd)
	devCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to listen on")
	serveCmd.Flags().StringVar(&provider, "provider", "openai", "AI provider to use (openai, anthropic, gemini)")
	serveCmd.Flags().StringVar(&model, "model", "gpt-4o-mini", "Model to use")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" && provider == "openai" {
		return fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	// Create provider with middleware
	var p core.Provider
	switch provider {
	case "openai":
		p = openai.New(
			openai.WithAPIKey(apiKey),
			openai.WithModel(model),
		)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	// Apply middleware
	p = middleware.Chain(
		middleware.WithRetry(middleware.RetryOpts{
			MaxAttempts: 3,
			BaseDelay:   time.Second,
			MaxDelay:    10 * time.Second,
			Jitter:      true,
		}),
		middleware.WithRateLimit(middleware.RateLimitOpts{
			RPS:   10,
			Burst: 20,
		}),
	)(p)

	// Set up HTTP routes
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/chat", handleChatSSE(p))
	mux.HandleFunc("/api/chat/ndjson", handleChatNDJSON(p))
	mux.HandleFunc("/api/generate", handleGenerate(p))
	mux.HandleFunc("/api/health", handleHealth)

	// Web interface
	mux.HandleFunc("/", handleWebInterface)

	// Start server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: logRequests(mux),
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-done
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("🚀 GAI Development Server started on http://localhost:%s", port)
	log.Printf("   Provider: %s, Model: %s", provider, model)
	log.Printf("   Press Ctrl+C to stop")

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func handleChatSSE(p core.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		messages := []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: req.Message}}},
		}

		if req.System != "" {
			messages = append([]core.Message{
				{Role: core.System, Parts: []core.Part{core.Text{Text: req.System}}},
			}, messages...)
		}

		s, err := p.StreamText(r.Context(), core.Request{
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			Stream:      true,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Stream error: %v", err), http.StatusInternalServerError)
			return
		}
		defer s.Close()

		// Use the normalized SSE handler
		config := stream.StreamConfig{
			RequestID: req.RequestID,
			Model:     model,
			Provider:  provider,
		}
		stream.SSENormalized(w, s, config)
	}
}

func handleChatNDJSON(p core.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		messages := []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: req.Message}}},
		}

		if req.System != "" {
			messages = append([]core.Message{
				{Role: core.System, Parts: []core.Part{core.Text{Text: req.System}}},
			}, messages...)
		}

		s, err := p.StreamText(r.Context(), core.Request{
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			Stream:      true,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Stream error: %v", err), http.StatusInternalServerError)
			return
		}
		defer s.Close()

		// Use the normalized NDJSON handler
		config := stream.StreamConfig{
			RequestID: req.RequestID,
			Model:     model,
			Provider:  provider,
		}
		stream.NDJSONNormalized(w, s, config)
	}
}

func handleGenerate(p core.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		messages := []core.Message{
			{Role: core.User, Parts: []core.Part{core.Text{Text: req.Message}}},
		}

		if req.System != "" {
			messages = append([]core.Message{
				{Role: core.System, Parts: []core.Part{core.Text{Text: req.System}}},
			}, messages...)
		}

		result, err := p.GenerateText(r.Context(), core.Request{
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Generation error: %v", err), http.StatusInternalServerError)
			return
		}

		resp := GenerateResponse{
			Text:  result.Text,
			Usage: result.Usage,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "healthy",
		"provider": provider,
		"model":    model,
		"version":  version,
	})
}

func handleWebInterface(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GAI Development Server</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            width: 100%;
            max-width: 800px;
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 {
            font-size: 2em;
            margin-bottom: 10px;
        }
        .header p {
            opacity: 0.9;
        }
        .chat-container {
            padding: 30px;
        }
        .input-group {
            margin-bottom: 20px;
        }
        .input-group label {
            display: block;
            margin-bottom: 5px;
            color: #555;
            font-weight: 500;
        }
        .input-group input, .input-group textarea {
            width: 100%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 14px;
            transition: border-color 0.3s;
        }
        .input-group input:focus, .input-group textarea:focus {
            outline: none;
            border-color: #667eea;
        }
        .button-group {
            display: flex;
            gap: 10px;
            margin-bottom: 20px;
        }
        button {
            flex: 1;
            padding: 12px 24px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            cursor: pointer;
            transition: transform 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
        }
        button:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        .response {
            background: #f5f5f5;
            border-radius: 8px;
            padding: 20px;
            min-height: 200px;
            max-height: 400px;
            overflow-y: auto;
            white-space: pre-wrap;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 14px;
        }
        .status {
            margin-top: 10px;
            padding: 10px;
            border-radius: 5px;
            background: #e3f2fd;
            color: #1976d2;
            font-size: 14px;
        }
        .error {
            background: #ffebee;
            color: #c62828;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 GAI Development Server</h1>
            <p>Provider: {{.Provider}} | Model: {{.Model}}</p>
        </div>
        <div class="chat-container">
            <div class="input-group">
                <label for="system">System Prompt (optional)</label>
                <textarea id="system" rows="2" placeholder="You are a helpful assistant..."></textarea>
            </div>
            <div class="input-group">
                <label for="message">Message</label>
                <textarea id="message" rows="3" placeholder="Enter your message here..."></textarea>
            </div>
            <div class="button-group">
                <button onclick="sendSSE()">Stream (SSE)</button>
                <button onclick="sendNDJSON()">Stream (NDJSON)</button>
                <button onclick="sendGenerate()">Generate</button>
            </div>
            <div class="response" id="response">Response will appear here...</div>
            <div class="status" id="status" style="display: none;"></div>
        </div>
    </div>

    <script>
        const responseEl = document.getElementById('response');
        const statusEl = document.getElementById('status');
        const systemEl = document.getElementById('system');
        const messageEl = document.getElementById('message');

        function showStatus(message, isError = false) {
            statusEl.textContent = message;
            statusEl.className = isError ? 'status error' : 'status';
            statusEl.style.display = 'block';
        }

        function clearResponse() {
            responseEl.textContent = '';
            statusEl.style.display = 'none';
        }

        async function sendSSE() {
            clearResponse();
            const message = messageEl.value.trim();
            if (!message) {
                showStatus('Please enter a message', true);
                return;
            }

            showStatus('Streaming with SSE...');
            
            try {
                const response = await fetch('/api/chat', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        message: message,
                        system: systemEl.value.trim(),
                        temperature: 0.7,
                        max_tokens: 1000
                    })
                });

                const reader = response.body.getReader();
                const decoder = new TextDecoder();
                let buffer = '';

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    buffer += decoder.decode(value, { stream: true });
                    const lines = buffer.split('\n');
                    buffer = lines.pop();

                    for (const line of lines) {
                        if (line.startsWith('data: ')) {
                            const data = line.slice(6);
                            if (data === '[DONE]') continue;
                            
                            try {
                                const event = JSON.parse(data);
                                if (event.type === 'text.delta' && event.text) {
                                    responseEl.textContent += event.text;
                                }
                            } catch (e) {
                                console.error('Parse error:', e);
                            }
                        }
                    }
                }
                showStatus('Stream completed');
            } catch (error) {
                showStatus('Error: ' + error.message, true);
            }
        }

        async function sendNDJSON() {
            clearResponse();
            const message = messageEl.value.trim();
            if (!message) {
                showStatus('Please enter a message', true);
                return;
            }

            showStatus('Streaming with NDJSON...');
            
            try {
                const response = await fetch('/api/chat/ndjson', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        message: message,
                        system: systemEl.value.trim(),
                        temperature: 0.7,
                        max_tokens: 1000
                    })
                });

                const reader = response.body.getReader();
                const decoder = new TextDecoder();
                let buffer = '';

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    buffer += decoder.decode(value, { stream: true });
                    const lines = buffer.split('\n');
                    buffer = lines.pop();

                    for (const line of lines) {
                        if (line.trim()) {
                            try {
                                const event = JSON.parse(line);
                                if (event.type === 'text.delta' && event.text) {
                                    responseEl.textContent += event.text;
                                }
                            } catch (e) {
                                console.error('Parse error:', e);
                            }
                        }
                    }
                }
                showStatus('Stream completed');
            } catch (error) {
                showStatus('Error: ' + error.message, true);
            }
        }

        async function sendGenerate() {
            clearResponse();
            const message = messageEl.value.trim();
            if (!message) {
                showStatus('Please enter a message', true);
                return;
            }

            showStatus('Generating...');
            
            try {
                const response = await fetch('/api/generate', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        message: message,
                        system: systemEl.value.trim(),
                        temperature: 0.7,
                        max_tokens: 1000
                    })
                });

                const data = await response.json();
                responseEl.textContent = data.text;
                showStatus('Generated. Tokens: ' + 
                    (data.usage ? data.usage.input_tokens + ' in, ' + data.usage.output_tokens + ' out' : 'N/A'));
            } catch (error) {
                showStatus('Error: ' + error.message, true);
            }
        }
    </script>
</body>
</html>`

	data := struct {
		Provider string
		Model    string
	}{
		Provider: provider,
		Model:    model,
	}

	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
		next.ServeHTTP(w, r)
	})
}

// ChatRequest represents a chat API request
type ChatRequest struct {
	Message     string  `json:"message"`
	System      string  `json:"system,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	RequestID   string  `json:"request_id,omitempty"`
}

// GenerateResponse represents a generate API response
type GenerateResponse struct {
	Text  string      `json:"text"`
	Usage core.Usage  `json:"usage"`
}