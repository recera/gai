# **Go AI Framework — Developer Experience & Usage Guide (DX-first)**

*A comprehensive, example‑driven reference for building with the `gai` library.*

> This document shows **how developers use the library** as if it were already published. It covers **every feature** we designed together: multi‑provider support (OpenAI, Anthropic, Gemini, Ollama, Groq/xAI/Baseten/Cerebras via OpenAI‑compatible), typed **structured outputs**, **tool calling** with multi‑step loops, **prompts** (embedded + overridable, versioned), **streaming** (SSE/NDJSON), **observability** (OpenTelemetry), **evals**, **model routing**, **memory & RAG**, **audio (TTS/STT)**, **citations & safety**, **file uploads**, and **MCP** (import & export).

> **Scope of this document:** Only the *developer API* and examples. Build plan, testing, and internals will follow in the next section of the project docs.

---

## Table of Contents

1. [Getting Started & Project Layout](#1-getting-started--project-layout)
2. [Providers at a Glance](#2-providers-at-a-glance)
3. [Messages & Parts (Text, Images, Audio, Video, Files)](#3-messages--parts-text-images-audio-video-files)
4. [Generate Text (single shot)](#4-generate-text-single-shot)
5. [Stream Text (rich event stream)](#5-stream-text-rich-event-stream)
6. [Structured Outputs (`GenerateObject[T]`, `StreamObject[T]`)](#6-structured-outputs-generateobjectt-streamobjectt)
7. [Typed Tools & Multi‑Step Loops (`StopWhen`)](#7-typed-tools--multi-step-loops-stopwhen)
8. [Audio (TTS & STT) + Built‑in `Speak` Tool](#8-audio-tts--stt--built-in-speak-tool)
9. [Citations & Safety Signals](#9-citations--safety-signals)
10. [Sessions, Safety, and Provider Options (Gemini‑first)](#10-sessions-safety-and-provider-options-gemini-first)
11. [Prompt Management (embedded, overrides, versions, fingerprints)](#11-prompt-management-embedded-overrides-versions-fingerprints)
12. [Streaming to Browsers (SSE & NDJSON)](#12-streaming-to-browsers-sse--ndjson)
13. [Observability (OpenTelemetry)](#13-observability-opentelemetry)
14. [Evaluations (programmatic)](#14-evaluations-programmatic)
15. [Model Routing (cost/latency/AB/failover)](#15-model-routing-costlatencyabfailover)
16. [Memory & RAG Helpers](#16-memory--rag-helpers)
17. [OpenAI‑Compatible Adapter (Groq, xAI, Baseten, Cerebras)](#17-openai-compatible-adapter-groq-xai-baseten-cerebras)
18. [File Uploads & Blob Refs](#18-file-uploads--blob-refs)
19. [MCP (Model Context Protocol) — Import & Export](#19-mcp-model-context-protocol--import--export)
20. [Error Handling, Retries, Rate Limits, Token Preflight](#20-error-handling-retries-rate-limits-token-preflight)

---

## 1) Getting Started & Project Layout

**Install** (module path is placeholder; substitute your actual module):

```bash
go get github.com/yourorg/ai
```

**Imports you’ll typically use:**

```go
import (
  "context"
  "github.com/yourorg/ai/core"
  "github.com/yourorg/ai/providers/openai"
  "github.com/yourorg/ai/providers/anthropic"
  "github.com/yourorg/ai/providers/gemini"
  "github.com/yourorg/ai/providers/ollama"
  compat "github.com/yourorg/ai/providers/openai_compat"
  "github.com/yourorg/ai/tools"
  "github.com/yourorg/ai/prompts"
  "github.com/yourorg/ai/stream"
  "github.com/yourorg/ai/media"        // TTS & STT
  "github.com/yourorg/ai/memory"
  "github.com/yourorg/ai/rag"
  "github.com/yourorg/ai/router"
  "github.com/yourorg/ai/mcp/client"
  "github.com/yourorg/ai/mcp/server"
)
```

**Top-level concepts** you’ll see throughout examples:

* `core.Provider` — unified interface implemented by each provider adapter.
* `core.Request` — one place to describe model, messages, tools, options, streaming, etc.
* `core.Message` & `core.Part` — multimodal content pieces (Text, ImageURL, Audio, Video, File).
* `tools.Handle` & `tools.New[I,O]` — **typed tools** with JSON Schema derived from Go types.
* `GenerateText`, `StreamText`, `GenerateObject[T]`, `StreamObject[T]`.
* `StopWhen` — multi‑step loop termination policy.
* `prompts.Registry` — versioned template rendering (with `//go:embed` + override).
* `stream.SSE/NDJSON` — stream providers’ events to browsers.
* `media.SpeechProvider / TranscriptionProvider` — ElevenLabs/Cartesia/Whisper/Deepgram.
* `mcp/client` & `mcp/server` — import/export tools/resources/prompts over MCP.

---

## 2) Providers at a Glance

Create a provider by calling its constructor with options:

```go
ctx := context.Background()

openAI := openai.New(
  openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
  openai.WithDefaultModel("gpt-4o-mini"),
)

anth := anthropic.New(
  anthropic.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
  anthropic.WithModel("claude-3-7-sonnet"),
)

g := gemini.New(
  gemini.WithAPIKey(os.Getenv("GOOGLE_API_KEY")),
  gemini.WithModel("gemini-1.5-pro"),
)

local := ollama.New(
  ollama.WithBaseURL("http://localhost:11434"),
  ollama.WithModel("llama3.2"),
)

// OpenAI-compatible (Groq, xAI, Baseten, Cerebras) — one adapter:
groq := compat.OpenAICompatible(compat.CompatOpts{
  BaseURL: "https://api.groq.com/openai/v1",
  APIKey:  os.Getenv("GROQ_API_KEY"),
})
```

You can swap providers **without changing** request/streaming/tool code.

---

## 3) Messages & Parts (Text, Images, Audio, Video, Files)

**Text-only message:**

```go
msg := core.Message{Role: core.User, Parts: []core.Part{
  core.Text{Text: "Summarize this article in 3 bullet points."},
}}
```

**Images by URL:**

```go
img := core.ImageURL{URL: "https://example.com/photo.jpg"}
msg := core.Message{Role: core.User, Parts: []core.Part{
  core.Text{Text: "Describe the scene."},
  img,
}}
```

**Audio/Video/File via BlobRef:**
The same `BlobRef` works for **URL**, **in‑memory bytes**, or a **provider file ID** (for systems like Gemini files API).

```go
aud := core.Audio{
  Source: core.BlobRef{
    Kind: core.BlobURL,
    URL:  "https://example.com/qna.wav",
    MIME: "audio/wav",
  },
}
vid := core.Video{Source: core.BlobRef{Kind: core.BlobURL, URL: "https://.../clip.mp4", MIME: "video/mp4"}}
doc := core.File{Source: core.BlobRef{Kind: core.BlobBytes, Bytes: pdfBytes, MIME: "application/pdf"}, Name: "paper.pdf"}

msg := core.Message{Role: core.User, Parts: []core.Part{
  core.Text{Text: "Transcribe and summarize this audio, then cite the doc."},
  aud, doc,
}}
```

---

## 4) Generate Text (single shot)

```go
res, err := openAI.GenerateText(ctx, core.Request{
  Messages: []core.Message{
    {Role: core.System, Parts: []core.Part{core.Text{"You are concise."}}},
    {Role: core.User, Parts: []core.Part{core.Text{"Give me 3 startup ideas about indoor farming."}}},
  },
  Temperature: 0.7,
  MaxTokens:   300,
})
if err != nil { log.Fatal(err) }

fmt.Println("Answer:", res.Text)
fmt.Printf("Tokens in/out: %d/%d\n", res.Usage.InputTokens, res.Usage.OutputTokens)
```

**Developer Notes**

* `GenerateText` can internally run **multi-step tool loops** if you pass tools and a `StopWhen`. If you have no tools, it’s a single shot.
* `System` prompts can be placed as the first message or passed via convenience `ProviderOptions` if the provider supports it.

---

## 5) Stream Text (rich event stream)

The event stream includes **text deltas**, **tool calls**, **tool results**, **citations**, **safety**, and **audio chunks** (if the provider emits audio).

```go
s, err := anth.StreamText(ctx, core.Request{
  Messages: []core.Message{{Role: core.User, Parts: []core.Part{
    core.Text{Text: "Explain transformers to a smart 8th grader."},
  }}},
  Stream: true,
})
if err != nil { log.Fatal(err) }
defer s.Close()

for ev := range s.Events() {
  switch ev.Type {
  case core.EventTextDelta:
    os.Stdout.WriteString(ev.TextDelta)
  case core.EventCitations:
    // Provider linked citations (e.g., Gemini)
    for _, c := range ev.Citations {
      log.Printf("Cited %s (%d-%d): %s", c.URI, c.Start, c.End, c.Title)
    }
  case core.EventSafety:
    log.Printf("Safety(%s): %s (%.2f)", ev.Safety.Category, ev.Safety.Action, ev.Safety.Score)
  case core.EventToolCall:
    log.Printf("Tool call: %s", ev.ToolName)
  case core.EventError:
    log.Printf("Stream error: %v", ev.Err)
  case core.EventFinish:
    fmt.Println("\n[done]")
  }
}
```

**Developer Notes**

* Streams are **backpressured** by channel capacity and respect `ctx` cancellation.
* You can export this stream directly to browsers via SSE (see §12).

---

## 6) Structured Outputs (`GenerateObject[T]`, `StreamObject[T]`)

Return **typed** JSON that conforms to your Go struct (we derive JSON Schema automatically).

```go
type Recipe struct {
  Name        string   `json:"name"`
  Ingredients []string `json:"ingredients"`
  Steps       []string `json:"steps"`
}

obj, err := g.GenerateObject[Recipe](ctx, core.Request{
  Messages: []core.Message{
    {Role: core.System, Parts: []core.Part{core.Text{"Return only valid JSON matching the schema."}}},
    {Role: core.User,   Parts: []core.Part{core.Text{"A vegetarian lasagna for 4 people"}}},
  },
})
if err != nil { log.Fatal(err) }

fmt.Println("Recipe:", obj.Value.Name)
```

**Developer Notes**

* Providers with **strict JSON** modes will be used automatically; others fall back to “JSON‑guardrails + validation + repair” if needed.
* `StreamObject[T]` mirrors stream semantics; you’ll receive progressive partial objects or end‑of‑stream validated object depending on provider capability.

---

## 7) Typed Tools & Multi‑Step Loops (`StopWhen`)

**Define a typed tool** with generic input/output:

```go
type WeatherIn struct{ Location string `json:"location"` }
type WeatherOut struct {
  Location string  `json:"location"`
  TempF    float64 `json:"temp_f"`
}

getWeather := tools.New[WeatherIn, WeatherOut](
  "get_weather", "Get the current temperature in a city",
  func(ctx context.Context, in WeatherIn, meta tools.Meta) (WeatherOut, error) {
    // Call your weather service; ctx is propagated
    return WeatherOut{Location: in.Location, TempF: 71.3}, nil
  },
)
```

**Use it in a multi‑step run** (the model decides when to call tools):

```go
res, err := openAI.GenerateText(ctx, core.Request{
  Messages: []core.Message{
    {Role: core.User, Parts: []core.Part{core.Text{"What's it like in San Francisco today?"}}},
  },
  Tools:     []tools.Handle{getWeather},
  ToolChoice: core.ToolAuto,
  StopWhen:   core.MaxSteps(4), // or other conditions below
})
if err != nil { log.Fatal(err) }
fmt.Println(res.Text)
```

**Stopping conditions** (compose as needed):

```go
// Stop after N steps
core.MaxSteps(4)

// Stop when assistant text matches a regex (e.g., "FINAL ANSWER:")
core.WhenTextMatches(regexp.MustCompile(`FINAL ANSWER:`))

// Stop as soon as a specific tool has run
core.UntilToolSeen("get_weather")

// Stop when no more tool calls are proposed after a step
core.NoMoreTools()
```

**Parallel tool calls**
If a step contains multiple tool calls, the runner executes them **concurrently** unless the provider disallows it. Results are appended to the message history before the next step.

---

## 8) Audio (TTS & STT) + Built‑in `Speak` Tool

### Text‑to‑Speech (TTS)

```go
tts := media.NewElevenLabs(
  media.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
  media.WithDefaultVoice("Rachel"),
)

speech, err := tts.Synthesize(ctx, media.SpeechRequest{
  Text:   "Hello from Go!",
  Voice:  "Rachel",
  Format: "mp3",
})
if err != nil { log.Fatal(err) }
defer speech.Close()

outFile := "hello.mp3"
f, _ := os.Create(outFile)
for chunk := range speech.Chunks() {
  f.Write(chunk)
}
f.Close()
log.Printf("Saved TTS to %s (format=%s)", outFile, speech.Format().MIME)
```

### Speech‑to‑Text (STT)

```go
stt := media.NewWhisper(
  media.WithBaseURL(os.Getenv("WHISPER_URL")),
)
tr, err := stt.Transcribe(ctx, media.TranscriptionRequest{
  Audio: core.BlobRef{Kind: core.BlobURL, URL: "https://example.com/question.wav", MIME: "audio/wav"},
})
if err != nil { log.Fatal(err) }
fmt.Println("Transcript:", tr.Text)
```

### Let the LLM **trigger** TTS via the `Speak` tool

```go
speak := media.NewSpeakTool(tts) // adapts your TTS provider into a typed tool

s, err := openAI.StreamText(ctx, core.Request{
  Messages: []core.Message{
    {Role: core.System, Parts: []core.Part{core.Text{"When you want to speak, call the speak tool."}}},
    {Role: core.User,   Parts: []core.Part{core.Text{"Say hello and introduce yourself."}}},
  },
  Tools:      []tools.Handle{speak},
  ToolChoice: core.ToolAuto,
  Stream:     true,
})
if err != nil { log.Fatal(err) }
defer s.Close()

for ev := range s.Events() {
  switch ev.Type {
  case core.EventToolResult:
    if ev.ToolName == "speak" {
      // Play the audio from ev.ToolResult (URL/FileID based on your tool impl)
      log.Printf("TTS result: %+v", ev.ToolResult)
    }
  case core.EventTextDelta:
    os.Stdout.WriteString(ev.TextDelta)
  case core.EventFinish:
    fmt.Println("\n[done]")
  }
}
```

---

## 9) Citations & Safety Signals

Some providers (e.g., Gemini) stream **citations** and **safety** feedback.

```go
s, _ := g.StreamText(ctx, core.Request{
  Messages: []core.Message{
    {Role: core.User, Parts: []core.Part{core.Text{"Give me a paragraph with sources about photosynthesis."}}},
  },
  Stream: true,
})

for ev := range s.Events() {
  switch ev.Type {
  case core.EventTextDelta:
    os.Stdout.WriteString(ev.TextDelta)
  case core.EventCitations:
    for _, c := range ev.Citations {
      fmt.Printf("\n[Source] %s — %s\n", c.URI, c.Title)
    }
  case core.EventSafety:
    fmt.Printf("\n[Safety] %s: %s (%.2f)\n", ev.Safety.Category, ev.Safety.Action, ev.Safety.Score)
  }
}
```

**Developer Notes**

* Citations can be token‑aligned with emitted spans; store them if you want clickable references in UIs.
* Safety events can be used to **stop** emission, redact, or degrade outputs.

---

## 10) Sessions, Safety, and Provider Options (Gemini‑first)

Gemini supports **cached content/sessions** and **safety** thresholds. The request shape handles this cleanly.

```go
res, err := g.GenerateText(ctx, core.Request{
  Messages: []core.Message{
    {Role: core.System, Parts: []core.Part{core.Text{"You are a helpful tutor."}}},
    {Role: core.User,   Parts: []core.Part{core.Text{"Explain entropy."}}},
  },
  Safety: &core.SafetyConfig{
    Harassment: "block_most",
    Sexual:     "block_some",
  },
  Session: &core.Session{
    Provider: "gemini",
    ID:       "cachedContent-abc123", // previously created or returned by adapter
  },
  ProviderOptions: map[string]any{
    "grounding": "web", // example; adapter maps/ignores as appropriate
  },
})
```

**Developer Notes**

* `ProviderOptions` is a flexible map for provider‑specific flags that we don’t model generically.
* `Session` can be used to reuse context (like cached content) to reduce tokens and latency.

---

## 11) Prompt Management (embedded, overrides, versions, fingerprints)

### Embed templates with `//go:embed`

```
prompts/
  summarize@1.2.0.tmpl
  tone_polite@0.1.0.tmpl
```

```go
//go:embed prompts/*.tmpl
var tmplFS embed.FS

reg := prompts.NewRegistry(
  tmplFS,
  prompts.WithOverrideDir(os.Getenv("PROMPTS_DIR")), // hot swap without rebuild
)

text, id, err := reg.Render(ctx, "summarize", "1.2.0", map[string]any{
  "Audience": "executives",
  "Length":   "short",
})
if err != nil { log.Fatal(err) }

req := core.Request{
  Messages: []core.Message{
    {Role: core.System, Parts: []core.Part{core.Text{text}}},
    {Role: core.User,   Parts: []core.Part{core.Text{"Paste article here..."}}},
  },
}
res, _ := openAI.GenerateText(ctx, req)

fmt.Println(res.Text)
```

**Developer Notes**

* Filenames carry **semver** (`name@MAJOR.MINOR.PATCH.tmpl`).
* The registry computes a content **fingerprint**; both `{name, version, fingerprint}` are attached to telemetry for auditability.
* You can keep prompts in source control and optionally override them via `PROMPTS_DIR` at runtime for quick testing.

---

## 12) Streaming to Browsers (SSE & NDJSON)

### Server (SSE)

```go
http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
  s, err := openAI.StreamText(r.Context(), core.Request{
    Messages: []core.Message{
      {Role: core.User, Parts: []core.Part{core.Text{r.URL.Query().Get("q")}}},
    },
  })
  if err != nil { http.Error(w, err.Error(), 500); return }
  defer s.Close()
  stream.SSE(w, s) // sets headers and flushes events
})

log.Fatal(http.ListenAndServe(":8080", nil))
```

### Client (browser)

```html
<script>
  const es = new EventSource("/api/chat?q=hello");
  es.onmessage = (e) => {
    const ev = JSON.parse(e.data);
    if (ev.type === "EventTextDelta") {
      document.querySelector("#out").textContent += ev.textDelta;
    }
    if (ev.type === "EventCitations") {
      console.log("Citations:", ev.citations);
    }
  };
</script>
<pre id="out"></pre>
```

**NDJSON** is similarly simple using `fetch()` + line splitting; use `stream.NDJSON(w, s)` server‑side.

---

## 13) Observability (OpenTelemetry)

The library starts spans **automatically** if OTel is configured. You can add custom attributes.

```go
ctx, span := obs.Tracer().Start(ctx, "my-handler")
defer span.End()

res, err := openAI.GenerateText(ctx, core.Request{
  Model: "gpt-4o-mini",
  Messages: []core.Message{{Role: core.User, Parts: []core.Part{core.Text{"hello"}}}},
  Metadata: map[string]any{"tenant":"acme-co"},
})
if err != nil {
  span.RecordError(err)
  log.Fatal(err)
}

span.SetAttributes(
  obs.String("app.tenant", "acme-co"),
  obs.Int("app.tokens_out", res.Usage.OutputTokens),
)
```

**What gets recorded by default**

* `llm.provider`, `llm.model`, `llm.temperature`, `llm.max_tokens`
* `usage.input_tokens`, `usage.output_tokens`, `usage.total_tokens`
* `prompt.name`, `prompt.version`, `prompt.fingerprint` (if using `prompts.Registry`)
* For tools: `tool.name`, latency, errors
* For audio: `tts.provider/voice/format/bytes`, `stt.provider/duration_ms`

---

## 14) Evaluations (programmatic)

Define an evaluator and score outputs:

```go
type ContainsEvaluator struct {
  Phrase string
}
func (c ContainsEvaluator) Name() string { return "contains" }
func (c ContainsEvaluator) Score(ctx context.Context, in evals.EvalInput) (map[string]float64, error) {
  score := 0.0
  if strings.Contains(strings.ToLower(in.Response.Text), strings.ToLower(c.Phrase)) {
    score = 1.0
  }
  return map[string]float64{"contains": score}, nil
}

runner := evals.NewRunner(
  evals.WithEvaluators(ContainsEvaluator{Phrase: "insight"}),
)

res, _ := openAI.GenerateText(ctx, core.Request{
  Messages: []core.Message{{Role: core.User, Parts: []core.Part{core.Text{"Share one key insight about Go."}}}},
})

scores, _ := runner.Score(ctx, evals.EvalInput{
  Request:  core.Request{ /* redacted or minimal clone */ },
  Response: *res,
})

fmt.Printf("Scores: %+v\n", scores)
```

You can also emit results to sinks (Braintrust/Phoenix/LangSmith) via configuration—no code changes.

---

## 15) Model Routing (cost/latency/AB/failover)

Pick the best provider/model automatically:

```go
rt := router.New(
  router.WithCandidate("openai:gpt-4o-mini", openAI),
  router.WithCandidate("anthropic:sonnet-3.7", anth),
  router.WithCandidate("groq:llama-3.3-70b", groq),
  router.WithPolicy(router.Policy{
    MaxLatencyMs: 1200,
    MaxCostUSD:   0.002,     // per request
    Prefer:       []string{"openai", "anthropic"}, // tie-breakers
    AB:           0.1,       // 10% explore
  }),
)

prov := rt.Select(ctx, core.Request{/* can use token estimate here */})
res, err := prov.GenerateText(ctx, /* same Request */)
```

**Developer Notes**

* The router can preflight **token estimates** (see §20) and respect **context windows**.
* A/B traffic emits attributes so your eval pipeline can compare.

---

## 16) Memory & RAG Helpers

### Simple chat memory (in‑memory)

```go
mem := memory.NewInMemory(100) // ring buffer of 100 messages

// Write conversation
mem.Append(core.Message{Role: core.User, Parts: []core.Part{core.Text{"Hello"}}})
mem.Append(core.Message{Role: core.Assistant, Parts: []core.Part{core.Text{"Hi, how can I help?"}}})

// Read last N
history := mem.Last(8)

// Use in a request
res, _ := openAI.GenerateText(ctx, core.Request{Messages: history})
```

### RAG with sqlite‑vec (local‑first)

```go
db, _ := sql.Open("sqlite3", "file:rag.db?_journal=WAL")
store := rag.SQLiteVec(db)
// 1) Embed documents (choose your embedding model/provider)
store.Index(ctx, rag.Documents{
  {ID: "doc1", Text: "Neural networks are universal approximators..."},
})

// 2) Query top-k
neighbors, _ := store.Search(ctx, "What is a universal approximator?", 3)

// 3) Compose prompt with citations
var ctxText string
for _, n := range neighbors {
  ctxText += fmt.Sprintf("\n[Source %s] %s", n.ID, n.Snippet)
}
req := core.Request{
  Messages: []core.Message{
    {Role: core.System, Parts: []core.Part{core.Text{"Use the provided context to answer."}}},
    {Role: core.User, Parts: []core.Part{core.Text{ctxText + "\n\nQuestion: What is it?"}}},
  },
}
res, _ := openAI.GenerateText(ctx, req)
```

---

## 17) OpenAI‑Compatible Adapter (Groq, xAI, Baseten, Cerebras)

Switch in any OpenAI‑compatible backend by changing the **base URL** and **key**:

```go
groq := compat.OpenAICompatible(compat.CompatOpts{
  BaseURL: "https://api.groq.com/openai/v1",
  APIKey:  os.Getenv("GROQ_API_KEY"),
})

xai := compat.OpenAICompatible(compat.CompatOpts{
  BaseURL: "https://api.x.ai/v1",
  APIKey:  os.Getenv("XAI_API_KEY"),
})

baseten := compat.OpenAICompatible(compat.CompatOpts{
  BaseURL: os.Getenv("BASETEN_OPENAI_URL"),
  APIKey:  os.Getenv("BASETEN_API_KEY"),
})

cerebras := compat.OpenAICompatible(compat.CompatOpts{
  BaseURL: "https://api.cerebras.ai/v1",
  APIKey:  os.Getenv("CEREBRAS_API_KEY"),
  DisableJSONStreaming:   true,  // example quirk
  DisableParallelToolCalls: true,
})
```

After that, your **requests, tools, streams, and prompts** work the same way.

---

## 18) File Uploads & Blob Refs

Some providers require **uploading files** (e.g., Gemini). Use the adapter’s `FileStore` helper or pass `BlobRef` with `Kind: BlobBytes` and let the adapter manage upload transparently.

```go
file := core.File{
  Source: core.BlobRef{
    Kind:  core.BlobBytes,
    Bytes: pdfBytes,
    MIME:  "application/pdf",
  },
  Name: "manual.pdf",
}

res, err := g.GenerateText(ctx, core.Request{
  Messages: []core.Message{
    {Role: core.User, Parts: []core.Part{
      core.Text{Text: "Extract the troubleshooting steps."},
      file,
    }},
  },
})
```

**Developer Notes**

* When a provider returns a **file ID**, the adapter may mutate the part to store `BlobRef{Kind: BlobProviderFile, FileID: "…"}` for subsequent calls in the same session.

---

## 19) MCP (Model Context Protocol) — Import & Export

### Import tools/resources/prompts from an MCP server

```go
mcpCli := mcpclient.New(
  mcpclient.WithStdioCmd("notes-server", "--root=./notes"),
  mcpclient.WithAllowTools("search_notes", "read_note"),
)
defer mcpCli.Close(ctx)

// Import their tools as []tools.Handle
handles, err := mcpCli.ImportTools(ctx)
if err != nil { log.Fatal(err) }

// Use in our loop:
res, err := openAI.GenerateText(ctx, core.Request{
  Messages: []core.Message{{Role: core.User, Parts: []core.Part{
    core.Text{"Find all notes about Rust borrow checker."},
  }}},
  Tools:      handles,
  ToolChoice: core.ToolAuto,
  StopWhen:   core.MaxSteps(3),
})
```

**Import server prompts into our registry:**

```go
reg := prompts.NewRegistry(embedFS)
if err := mcpCli.SyncPrompts(ctx, reg); err != nil { log.Fatal(err) }
text, id, _ := reg.Render(ctx, "notes/summarize", "", map[string]any{"topic":"ownership"})
```

### Export your project as an MCP server

```go
srv := mcpserver.New(
  mcpserver.WithTools(getWeather /*, others */),
  mcpserver.WithResources(mcpserver.FSResource("/data/reports")),
  mcpserver.WithPrompts(reg),
)

// Local (Claude Desktop, etc.)
go srv.ServeStdio(ctx)

// Remote
go srv.ServeHTTP(ctx, ":8765", mcpserver.WithBearerAuth(os.Getenv("TOKEN")))
```

---

## 20) Error Handling, Retries, Rate Limits, Token Preflight

**Retry with backoff** (idempotent operations only):

```go
prov := middleware.WithRetry(openAI, middleware.RetryOpts{
  MaxAttempts: 4,
  BaseDelay:   100 * time.Millisecond,
  MaxDelay:    2 * time.Second,
  Jitter:      true,
})
```

**Rate limit** per provider:

```go
prov = middleware.WithRateLimit(prov, middleware.RateLimitOpts{
  RPS: 10, Burst: 20,
})
```

**Token preflight & context window checks**:

```go
est := core.EstimateTokens(core.Request{
  Model: "gpt-4o-mini",
  Messages: []core.Message{
    {Role: core.User, Parts: []core.Part{core.Text{bigText}}}, // big doc
  },
})
if est.Input > 120000 { // hypothetical context window
  chunks := core.SplitForContextWindow(bigText, 80000)
  // stream a summarize-per-chunk pipeline
}
```

**Error taxonomy**:

* `aierrors.IsRateLimited(err)`
* `aierrors.IsContentFiltered(err)`
* `aierrors.IsTransient(err)`
* `aierrors.IsBadRequest(err)`

---

# End-to-end Examples

Below are a few **complete** scenarios stitching features together.

---

### Example A: **Enterprise summarization API** (prompts + SSE + OpenTelemetry)

```go
//go:embed prompts/*.tmpl
var tmplFS embed.FS

func main() {
  openAI := openai.New(openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
  reg := prompts.NewRegistry(tmplFS, prompts.WithOverrideDir("./overrides"))

  http.HandleFunc("/api/summarize", func(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    text := r.FormValue("text")
    orient := r.FormValue("audience") // e.g., "exec"

    sys, id, err := reg.Render(ctx, "summarize", "1.2.0", map[string]any{
      "Audience": orient,
      "Length":   "short",
    })
    if err != nil { http.Error(w, err.Error(), 500); return }

    s, err := openAI.StreamText(ctx, core.Request{
      Messages: []core.Message{
        {Role: core.System, Parts: []core.Part{core.Text{sys}}},
        {Role: core.User,   Parts: []core.Part{core.Text{text}}},
      },
      Stream: true,
      Metadata: map[string]any{
        "prompt.name": id.Name, "prompt.version": id.Version, "prompt.fp": id.Fingerprint,
      },
    })
    if err != nil { http.Error(w, err.Error(), 500); return }
    defer s.Close()

    stream.SSE(w, s)
  })

  log.Fatal(http.ListenAndServe(":8080", nil))
}
```

---

### Example B: **Voice agent** (STT → Gemini reasoning → TTS via Speak tool)

```go
stt := media.NewDeepgram(media.WithAPIKey(os.Getenv("DEEPGRAM_API_KEY")))
tts := media.NewCartesia(media.WithAPIKey(os.Getenv("CARTESIA_API_KEY")))
speak := media.NewSpeakTool(tts)
g := gemini.New(gemini.WithAPIKey(os.Getenv("GOOGLE_API_KEY")), gemini.WithModel("gemini-1.5-flash"))

audio := core.Audio{Source: core.BlobRef{Kind: core.BlobURL, URL: "https://example.com/user.wav", MIME: "audio/wav"}}

s, err := g.StreamText(ctx, core.Request{
  Messages: []core.Message{
    {Role: core.System, Parts: []core.Part{core.Text{"Be a helpful voice concierge."}}},
    {Role: core.User, Parts: []core.Part{audio}},
  },
  Tools:      []tools.Handle{speak},
  ToolChoice: core.ToolAuto,
  Stream:     true,
})
if err != nil { log.Fatal(err) }
defer s.Close()

for ev := range s.Events() {
  switch ev.Type {
  case core.EventTextDelta:
    fmt.Print(ev.TextDelta) // live transcript/responses
  case core.EventToolResult:
    if ev.ToolName == "speak" {
      // Retrieve audio URL/FileID from the tool result and play
    }
  }
}
```

---

### Example C: **RAG with MCP resources & OpenAI‑compatible Groq**

```go
groq := compat.OpenAICompatible(compat.CompatOpts{
  BaseURL: "https://api.groq.com/openai/v1",
  APIKey: os.Getenv("GROQ_API_KEY"),
})

mcpCli := mcpclient.New(mcpclient.WithStdioCmd("docs-mcp", "--root=./docs"))
defer mcpCli.Close(ctx)

// Fetch remote resources (URIs) and index locally
uris, _ := mcpCli.ListResourceURIs(ctx, "docs:")
var docs []rag.Document
for _, u := range uris {
  b, _ := mcpCli.ReadResource(ctx, u)
  docs = append(docs, rag.Document{ID: u, Text: string(b)})
}

db, _ := sql.Open("sqlite3", "file:rag.db")
vec := rag.SQLiteVec(db)
vec.Index(ctx, docs)

neighbors, _ := vec.Search(ctx, "How do transactions work?", 3)
var context string
for _, n := range neighbors {
  context += "\n" + n.Snippet
}

res, _ := groq.GenerateText(ctx, core.Request{
  Messages: []core.Message{
    {Role: core.System, Parts: []core.Part{core.Text{"Answer using only the provided docs context."}}},
    {Role: core.User, Parts: []core.Part{core.Text{context + "\n\nQuestion: Explain transactions."}}},
  },
})
fmt.Println(res.Text)
```

---

### Example D: **Agent with imported MCP tool + local tool; routed provider**

```go
rt := router.New(/* candidates + policy */)
prov := rt.Select(ctx, core.Request{})

mcpCli := mcpclient.New(mcpclient.WithStdioCmd("calendar-mcp"))
defer mcpCli.Close(ctx)

mcpTools, _ := mcpCli.ImportTools(ctx) // e.g., "get_calendar", "create_event"

type CalcIn struct{ Expr string `json:"expr"` }
type CalcOut struct{ Result float64 `json:"result"` }
calc := tools.New[CalcIn, CalcOut]("calc", "Evaluate an arithmetic expression", func(ctx context.Context, in CalcIn, _ tools.Meta) (CalcOut, error) {
  v, err := eval(in.Expr) // your code
  if err != nil { return CalcOut{}, err }
  return CalcOut{Result: v}, nil
})

res, err := prov.GenerateText(ctx, core.Request{
  Messages: []core.Message{
    {Role: core.System, Parts: []core.Part{core.Text{"You can do math via 'calc' and manage events via calendar tools."}}},
    {Role: core.User,   Parts: []core.Part{core.Text{"Schedule my workout for 7am tomorrow and tell me how many minutes between 7am and 8:15am."}}},
  },
  Tools:      append([]tools.Handle{calc}, mcpTools...),
  ToolChoice: core.ToolAuto,
  StopWhen:   core.MaxSteps(6),
})
if err != nil { log.Fatal(err) }
fmt.Println(res.Text)
```

---

## Closing Notes (DX)

* **Everything is Go‑idiomatic**: `context.Context` for cancellation & tracing; channels for streams; generics for typed outputs & tools.
* **Swap providers without refactors**; move from local Ollama to OpenAI to Gemini by changing the provider constructor.
* **Prompts are first‑class code** with versions and fingerprints; override at runtime without a rebuild.
* **Observability is on by default**; add your attributes if you want.
* **Evals are easy**; run in CI or locally; export to your favorite sink.
* **MCP makes your app interoperable**, both as a client (import external tools/resources/prompts) and as a server (expose your own).

---

> Next section (in a separate document): **Engineering Build Plan & Internals** — repository layout, implementation details for each package, testing strategy (unit/integration/e2e), CI/CD, performance budgets, conformance matrices, and contribution guidelines.
