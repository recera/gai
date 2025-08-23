# Go AI Framework — **Comprehensive Build Plan & Architecture**

> This is the **production-grade** engineering build plan for the `gai` library we designed. It’s written to be consumed by human developers and/or capable coding agents working **in parallel**. It contains:
>
> * Architecture & design details (APIs, packages, invariants)
> * A phased roadmap with **parallelizable workstreams**
> * Implementation notes, code scaffolds, and acceptance criteria
> * Unit/integration/e2e/perf test plans for each phase
> * Security, reliability, and release engineering requirements

**Go version:** 1.22+
**License:** Apache-2.0 (or MIT) — pick one and stick to it
**Target platforms:** Linux, macOS, Windows
**No cgo required** (unless a specific optional module demands it)

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Core Design Principles](#2-core-design-principles)
3. [Repository Layout & Ownership](#3-repository-layout--ownership)
4. [Cross-Cutting Concerns & Standards](#4-cross-cutting-concerns--standards)
5. [Phased Build Plan (with Parallel Workstreams)](#5-phased-build-plan-with-parallel-workstreams)
6. [Detailed Package Specs](#6-detailed-package-specs)
7. [Provider Adapters (Canonical Requirements)](#7-provider-adapters-canonical-requirements)
8. [Middleware (Retry/RateLimit/Safety/Caching)](#8-middleware-retryratelimitsafetycaching)
9. [Prompt System (Embedded + Overrides + Versions)](#9-prompt-system-embedded--overrides--versions)
10. [Tools & JSON Schema Reflection](#10-tools--json-schema-reflection)
11. [Streaming (SSE/NDJSON) & Browser Integration](#11-streaming-ssendjson--browser-integration)
12. [Observability (OpenTelemetry) & Costs](#12-observability-opentelemetry--costs)
13. [Evaluations & Sinks](#13-evaluations--sinks)
14. [Audio (TTS/STT) & Built-in Speak Tool](#14-audio-ttsstt--built-in-speak-tool)
15. [OpenAI-Compatible Adapter (Groq/xAI/Baseten/Cerebras)](#15-openai-compatible-adapter-groqxaibasetencerebras)
16. [Gemini-Specific Features (Files, Safety, Citations)](#16-gemini-specific-features-files-safety-citations)
17. [MCP Support (Client & Server)](#17-mcp-support-client--server)
18. [Memory & RAG Helpers](#18-memory--rag-helpers)
19. [Router (Cost/Latency/A-B/Failover)](#19-router-costlatencyabfailover)
20. [Token Estimation & Context Window Utilities](#20-token-estimation--context-window-utilities)
21. [Testing Strategy & Matrices](#21-testing-strategy--matrices)
22. [Security, Privacy & Compliance](#22-security-privacy--compliance)
23. [Docs, Examples, CLI & Website](#23-docs-examples-cli--website)
24. [CI/CD, Release Engineering & Versioning](#24-cicd-release-engineering--versioning)
25. [Contribution Guidelines & Governance](#25-contribution-guidelines--governance)

---

## 1) Architecture Overview

```
/ai
  /core         <- request/response types, messages/parts, events, runner, errors
  /providers    <- openai, anthropic, gemini, ollama, openai_compat, (subpackages)
  /tools        <- typed tools (Tool[I,O]), JSON Schema reflection
  /prompts      <- versioned templates (embed + overrides), fingerprints
  /stream       <- SSE + NDJSON adapters for TextStream/ObjectStream
  /obs          <- OpenTelemetry tracing/metrics, usage & cost accounting
  /evals        <- evaluators, datasets, sinks (braintrust, phoenix, langsmith)
  /middleware   <- retry/backoff, rate limiting, safety, (optional) caching
  /media        <- TTS/STT providers (ElevenLabs, Cartesia, Whisper/Deepgram)
  /file         <- FileStore & BlobRef helpers, temp store implementations
  /mcp          <- client (import), server (export), transports, policy, codegen
  /memory       <- chat history stores (in-mem, sqlite)
  /rag          <- minimal vector adapters (sqlite-vec, pgvector)
  /router       <- provider/model routing, AB, failover
  /cmd/ai       <- CLI: dev server, eval runner, prompt tools, codegen
  /x            <- experimental features behind build tags
```

**Dependency rules**

* `core` has no deps on other internal packages (except stdlib).
* `providers/*` depend on `core`, `tools`, `obs`, `middleware`, `file` as needed; no provider depends on another provider.
* `tools` depends on a single schema reflection library (e.g., `invopop/jsonschema`) — isolate behind a narrow facade.
* `stream`, `prompts`, `obs`, `file` depend only on `core` (and minimal libs).
* `mcp` may depend on `tools`, `prompts`, `obs`.
* `media` depends only on stdlib + HTTP; isolate any heavy SDKs behind local lightweight clients.

**Concurrency model**

* `context.Context` is **mandatory** on all public API calls.
* Streaming uses a **channel** (`<-chan Event`) for backpressure; optionally an `io.Reader` NDJSON view for piping.
* Multi-step tool loop spawns **goroutines** to run parallel tool calls; results are merged deterministically.

---

## 2) Core Design Principles

1. **Go-first ergonomics:** typed APIs, generics, `context`, channels, `io.Reader`; no `any` in public API unless interoperability demands it.
2. **Provider-agnostic abstractions:** one `Provider` interface; adapters normalize capabilities and stream events to common enums.
3. **Typed structured outputs:** `GenerateObject[T]` drives JSON Schema from Go types; strict modes when supported; robust fallback validation.
4. **Tools as code:** typed `Tool[I,O]`, compile-time schema generation; multi-step loops controlled by `StopWhen` conditions.
5. **Extensibility sealed at edges:** `ProviderOptions` and pluggable sinks/adapters without leaking provider internals into core.
6. **Observability by default:** traces/metrics on every call, step, and tool; opt-out via config, not code.
7. **Single-binary deploy:** prompts embed + runtime override; no Node/Python required.
8. **Security first:** allowlists, quotas, timeouts for tool execution; content-size caps; transport auth for MCP HTTP.
9. **Compatibility & stability:** semantic versioning; crisp JSON-compat contracts; CI checks for prompt version bumps.

---

## 3) Repository Layout & Ownership

**Owners (CODEOWNERS):**

* `/core`, `/tools`, `/prompts`, `/stream`: **Core team**
* `/providers/*`: **Provider squads** (one squad per adapter)
* `/obs`, `/evals`: **Telemetry squad**
* `/mcp/*`: **MCP squad**
* `/media/*`: **Media squad**
* `/middleware`, `/router`: **Runtime squad**
* `/rag`, `/memory`, `/file`: **Data infra squad**
* `/cmd/ai`, `/docs`: **DX squad**

**Branching**

* `main` protected; PRs require tests + linters + review.
* `feature/*` branches per squad; frequent merges to main behind shippable slices.

---

## 4) Cross-Cutting Concerns & Standards

* **Coding style:** `gofmt`, `go vet`, `staticcheck`; idiomatic naming; exported symbols well-documented.
* **Errors:** sentinel helpers `aierrors.IsRateLimited(err)`, `IsTransient`, `IsContentFiltered`, `IsBadRequest`.
* **Timeouts:** default per provider (e.g., 60s), override via context deadlines and options.
* **HTTP:** shared `*http.Client` with tuned `Transport` (keep-alives, `MaxIdleConns`, timeouts), gzip if supported.
* **Allocations:** avoid per-chunk heap allocations; `sync.Pool` for buffers; zero-copy where practical.
* **Thread safety:** providers are safe for concurrent use; document any per-instance config immutability.
* **Dependency policy:** prefer stdlib; limit third-party libs; pin versions (renovate/bot for bumps); run `govulncheck`.

---

## 5) Phased Build Plan (with Parallel Workstreams)

> Each phase lists **scope**, **deliverables**, **parallel tracks**, **dependencies**, **tests**, and **acceptance criteria**. Squads can run tracks in parallel where noted.

### **Phase 0 — Repo Bootstrap (1–2 days)**

**Scope**

* Init `go.mod`, license, `README`, `CONTRIBUTING`, `CODE_OF_CONDUCT`, `SECURITY.md`.
* CI with GitHub Actions: lint, vet, staticcheck, unit tests (race), coverage, `govulncheck`.
* Pre-commit hooks (`golangci-lint` optional).

**Deliverables**

* Repo skeleton with empty packages and stubs.
* GitHub labels, issue/PR templates.
* CODEOWNERS.

**Tests**

* CI runs (empty package tests pass).
* Lint pipeline green.

**Acceptance**

* CI green; basic docs render; license header on files.

---

### **Phase 1 — Core & Stream Primitives (parallelizable)**

**Tracks**

1. **core/types**

   * `Message`, `Part` (Text, ImageURL, Audio, Video, File), `BlobRef`
   * `Request`, `Usage`, `Step`, `TextResult`, `ObjectResult[T]`
   * `EventType`, `Event`, `TextStream` interface
   * `StopCondition` & helpers (`MaxSteps`, `WhenTextMatches`, `UntilToolSeen`, `NoMoreTools`)
   * Error taxonomy package `aierrors`

2. **core/runner**

   * Multi-step loop engine: detect tool calls, execute in parallel, merge results, stop on conditions

3. **stream package**

   * SSE writer: headers, heartbeat, flush, error handling
   * NDJSON writer

**Dependencies**: None (stdlib only)

**Unit Tests**

* JSON marshal/unmarshal of parts & requests.
* Runner step logic (single tool, multiple tools, stop conditions).
* SSE writer behavior (headers, heartbeat, chunking) using `httptest`.
* NDJSON writer line framing.

**Acceptance**

* 90%+ coverage in `core`/`stream`.
* Data races test pass; goroutine leaks detection in runner tests.

---

### **Phase 2 — Tools & Schema Reflection (parallelizable)**

**Tracks**

1. **tools core**

   * `Tool[I,O]`, `Handle`, `Meta`, `New`
   * Input/output schema generation via `invopop/jsonschema` (isolated in a tiny adapter)
   * Exec path: decode `json.RawMessage` → `I`, run `Execute`, encode `O`

2. **Validation & utilities**

   * Schema cache keyed by reflect.Type
   * Runtime JSON validation helper for fallback providers

**Dependencies**: `core` only

**Unit Tests**

* Schema generation for nested structs, maps, arrays, enums (custom tags).
* Exec decode/encode round-trip; invalid inputs yield errors.

**Acceptance**

* Schema reflection stable across Go versions; golden JSON snapshots checked in.

---

### **Phase 3 — Prompts System (embedded + overrides + fingerprints)**

**Tracks**

1. **prompts/registry**

   * `NewRegistry(embed.FS, opts...)` with `WithOverrideDir`
   * Render via `text/template` with helper funcs (indent, join, json)
   * Versioned filenames (`name@semver.tmpl`); compute SHA-256 fingerprint
   * `TemplateID{Name, Version, Fingerprint}` return value

2. **CI utility**

   * `cmd/ai prompts bump/verify` (optional now, or in CLI phase)
   * Verify “version bump if content changed”

**Unit Tests**

* Render with/without override; fingerprint stability.
* Template helpers.

**Acceptance**

* Deterministic render and fingerprint; override precedence works.

---

### **Phase 4 — Observability (OTel) & Usage/Cost**

**Tracks**

1. **obs/tracing**

   * Span creation helpers for request, step, tool
   * Attributes: provider/model/temperature/max\_tokens, usage tokens, prompt ids

2. **obs/metrics**

   * Histograms for latency; counters for errors; gauges for tokens if useful

3. **Usage accounting**

   * Helper to collect usage from provider responses
   * Fallback token estimation hooks (wired later in Phase 13)

**Unit Tests**

* Use an in-memory OTel exporter to assert spans/attributes.
* Verify spans nest correctly across runner steps.

**Acceptance**

* Automatic tracing when global tracer is present; zero overhead when absent.

---

### **Phase 5 — Provider: OpenAI (Canonical)**

**Tracks**

1. **HTTP client**

   * Shared `*http.Client` with tuned transport; auth header; retry on 429/5xx (basic)

2. **GenerateText / StreamText**

   * Responses API (preferred); fall back to Chat Completions if env opt-in
   * Streaming normalization → `Event` enums (text/tool/error/finish)

3. **GenerateObject\[T] / StreamObject\[T]**

   * Use OpenAI Strict Structured Outputs (JSON Schema) when available
   * Fallback to guarded JSON + validation

4. **Tools**

   * Map our `Tool` to OpenAI tools/functions; parse tool calls; parallel exec; multi-step loop integration

**Integration Tests**

* Against a **mock OpenAI server** (httptest) with canned fixtures for stream & tools.
* Optional live tests (skipped unless `OPENAI_API_KEY` present).

**Acceptance**

* Streaming, tools, structured outputs work against fixtures; live smoke tests pass.

---

### **Phase 6 — Middleware (Retry/RateLimit/Safety)**

**Tracks**

1. **retry/backoff**

   * Configurable attempts, jitter, max delay, idempotency guard

2. **ratelimit**

   * Token bucket (`x/time/rate`) per provider instance; header-based hints future-proofed

3. **safety**

   * Stream transform hook to redact/stop; blocklists/regex; PII placeholders

**Unit Tests**

* Retry on transient, not on 4xx; jitter within bounds.
* Rate limit passes bursts then smooths; concurrency safe.
* Safety transform redacts expected patterns; early-stop works.

**Acceptance**

* Middleware composable around any Provider.

---

### **Phase 7 — Stream to Browser (Server helpers)**

**Tracks**

1. **stream.SSE HTTP handler**

   * Set `Content-Type`, `Cache-Control`, `Connection`, heartbeat `: keep-alive`
   * Flush on each event; handle errors

2. **NDJSON handler**

**Integration Tests**

* `httptest.Server` + EventSource polyfill test (or simplified); verify message order.
* NDJSON line integrity tests.

**Acceptance**

* Works with modern browsers and proxies.

---

### **Phase 8 — CLI (minimal) & Examples (hello world)**

**Tracks**

1. **cmd/ai** (initial)

   * `ai dev serve` to start a minimal SSE endpoint using OpenAI adapter
   * `ai prompts verify` (optional)

2. **examples/**

   * Hello text, Hello stream, Hello object, Hello tool

**Acceptance**

* Examples compile and run; CLI starts server with environment variables.

---

### **Phase 9 — Provider: Anthropic**

**Tracks**

1. **SSE stream mapping**

   * Map Anthropic event taxonomy to `Event` enums; reassemble tool input JSON deltas

2. **GenerateText / StreamText / Tools**

   * Messages API; tool use; steps integrate with runner

3. **Usage**

   * Collect usage if provided; else estimate later

**Integration Tests**

* Mock SSE server; deltas with tool\_use; malformed chunk tests.
* Live smoke behind `ANTHROPIC_API_KEY`.

**Acceptance**

* Stable streaming; multi-step tools; errors normalized.

---

### **Phase 10 — Provider: Gemini (Files/Safety/Citations)**

**Tracks**

1. **Files API & BlobRef**

   * Upload helper; map `BlobRef` → `file_data/inline_data`; store `BlobProviderFile` IDs.

2. **Safety & Citations**

   * Map safety threshold options; emit `EventSafety`; pass-through citations as `EventCitations`.

3. **Structured outputs**

   * Map `GenerateObject[T]` to Gemini response\_schema; fallback consistent.

**Integration Tests**

* Mock Gemini HTTP server; file upload; citation streaming; safety events.
* Live smoke behind `GOOGLE_API_KEY`.

**Acceptance**

* Multimodal requests with audio/video/files; citations & safety reachable on stream.

---

### **Phase 11 — Provider: Ollama (Local + Tools)**

**Tracks**

1. **Local HTTP client**

   * Streaming normalization; tool calling mapping (when supported)

2. **Model capabilities**

   * Basic catalog retrieval; handle offline errors gracefully

**Integration Tests**

* Requires local Ollama; mark as **optional** gated tests.
* Mock server for CI default.

**Acceptance**

* Developer iteration flow works locally.

---

### **Phase 12 — OpenAI-Compatible Adapter (Groq/xAI/Baseten/Cerebras)**

**Tracks**

1. **Compat core**

   * `CompatOpts{BaseURL, APIKey, DisableJSONStreaming, DisableParallelToolCalls, UnsupportedParams, PreferResponsesAPI}`
   * Capability probe `/v1/models` or equivalent

2. **Quirks registry**

   * Hostname → defaults (e.g., Cerebras: disable JSON-mode streaming, parallel\_tool\_calls off by default)

3. **Presets**

   * `presets.Groq()`, `presets.XAI()`, `presets.Baseten(url)`, `presets.Cerebras()`

**Tests**

* Mock servers per quirk; ensure params trimmed; streaming works.
* Live smoke optional if keys present.

**Acceptance**

* One codepath supports all four; documented quirks respected.

---

### **Phase 13 — Token Estimation & Context Window Utilities**

**Tracks**

1. **Tokenizer**

   * Pure-Go tiktoken port or compatible; model configs (approximate is fine initially)

2. **Preflight**

   * Estimate tokens for request; utilities to split content to fit window

**Unit Tests**

* Known token count fixtures for a subset of models.
* Splitting utilities preserve boundaries cleanly.

**Acceptance**

* Router and apps can rely on estimates for selection and chunking.

---

### **Phase 14 — Audio (TTS/STT) & Speak Tool**

**Tracks**

1. **media.SpeechProvider** adapters

   * ElevenLabs, Cartesia minimal HTTP clients; streaming chunk parsing; format descriptors

2. **media.TranscriptionProvider**

   * Whisper (server), Deepgram minimal HTTP clients

3. **tools.Speak**

   * `SpeakIn{Text,Voice,Format}`, `SpeakOut{URL|FileID}`; saves chunks to temp file or returns data URL (configurable)

**Integration Tests**

* Mock servers; audio chunk flows; format detection.
* Live smoke optional behind keys.

**Acceptance**

* Voice agent example runs end-to-end (mocked by default, real optional).

---

### **Phase 15 — Evaluations & Sinks**

**Tracks**

1. **evals core**

   * `Evaluator` interface; `EvalInput`/`EvalResult`; `Runner`

2. **Built-in evaluators**

   * exact match, regex/contains, JSON schema validate, LLM-as-judge (provider-agnostic)

3. **Sinks**

   * Braintrust (experiments), Phoenix/Arize, LangSmith — minimal REST clients

4. **cmd/ai eval run**

   * dataset ingestion (CSV/JSONL), runner pipeline, sink publish

**Tests**

* Evaluator correctness; dataset iteration; sink posting (mock HTTP).
* CLI e2e with small dataset.

**Acceptance**

* Users can run offline evals and push results to a sink with one command.

---

### **Phase 16 — MCP (Client & Server)**

**Tracks**

1. **mcp/client**

   * stdio & streamable HTTP transports; JSON-RPC framing
   * `ImportTools` (dynamic `tools.Handle`), `ReadResource`, `SyncPrompts`

2. **mcp/server**

   * Expose our tools/resources/prompts via stdio & HTTP
   * Auth for HTTP transport; policy allowlists/quotas/timeouts

3. **Codegen (optional)**

   * `ai mcp codegen` to generate typed Go wrappers from remote tool schemas

**Integration Tests**

* Fake server & client cross-tests; round-trip calls; resource read; prompts list/get.
* Subprocess stdio loop test.

**Acceptance**

* Import/Export demonstrated in examples; policy defaults safe.

---

### **Phase 17 — Memory & RAG Helpers**

**Tracks**

1. **memory**

   * In-memory ring buffer; sqlite-backed persistent store

2. **rag**

   * sqlite-vec adapter; chunking; basic embed API adapter (pluggable provider)

**Tests**

* sqlite-vec indexing/search; chunk boundaries; persistent memory compaction.

**Acceptance**

* RAG example runs with sqlite-vec locally.

---

### **Phase 18 — Router (Cost/Latency/A‑B/Failover)**

**Tracks**

1. **Policy**

   * `MaxLatencyMs`, `MaxCostUSD`, provider/model allowlist/prefer, AB percentage

2. **Selector**

   * Token preflight; context window; heuristics for selection; fallback on rate-limit/overload

3. **Telemetry**

   * Emit attributes for experiment id & variant

**Tests**

* Deterministic selection given inputs; AB randomness within epsilon; failover path correct.

**Acceptance**

* Router used transparently in examples; docs complete.

---

### **Phase 19 — Docs, Samples, Website & Polishing**

**Tracks**

1. **Docs site** (mkdocs or docusaurus) — build from `/docs`
2. **Samples**: web SSE app, voice agent, RAG demo, MCP integration, router demo
3. **Goreleaser** for CLI binaries (optional early release)
4. **API reference** (godoc + examples)

**Acceptance**

* Docs build, samples compile, quickstart path verified.

---

## 6) Detailed Package Specs

Below are condensed specs and key signatures.

### `core`

```go
type Role string
const (System Role = "system"; User = "user"; Assistant = "assistant"; Tool = "tool")

type Part interface{ isPart() }
type Text struct{ Text string }        func (Text) isPart(){}
type ImageURL struct{ URL string }     func (ImageURL) isPart(){}
type BlobKind uint8
const (BlobURL BlobKind = iota; BlobBytes; BlobProviderFile)
type BlobRef struct{ Kind BlobKind; URL string; Bytes []byte; FileID, MIME string; Size int64 }
type Audio struct{ Source BlobRef; SampleRate, Channels int } func (Audio) isPart(){}
type Video struct{ Source BlobRef }      func (Video) isPart(){}
type File  struct{ Source BlobRef; Name, Purpose string } func (File) isPart(){}

type Message struct {
  Role  Role
  Parts []Part
  Name  string // optional
}

type ToolChoice int
const (ToolAuto ToolChoice = iota; ToolNone; ToolRequired; ToolSpecific)

type SafetyConfig struct{ Harassment, Hate, Sexual, Dangerous string }
type Session struct{ Provider, ID string }

type Request struct {
  Model string
  Messages []Message
  Temperature float32
  MaxTokens   int
  Tools []tools.Handle
  ToolChoice ToolChoice
  StopWhen   StopCondition
  Safety     *SafetyConfig
  Session    *Session
  ProviderOptions map[string]any
  Metadata   map[string]any
  Stream     bool
}

type Usage struct{ InputTokens, OutputTokens, TotalTokens int }

type ToolCall struct{ Name string; Input json.RawMessage }
type ToolExecution struct{ Name string; Result any }

type Step struct {
  Text        string
  ToolCalls   []ToolCall
  ToolResults []ToolExecution
}
type TextResult struct{ Text string; Steps []Step; Usage Usage; Raw any }
type ObjectResult[T any] struct{ Value T; Steps []Step; Usage Usage; Raw any }

type EventType int
const (
  EventStart EventType = iota
  EventTextDelta
  EventAudioDelta
  EventToolCall
  EventToolResult
  EventCitations
  EventSafety
  EventFinishStep
  EventFinish
  EventError
  EventRaw
)

type AudioFormat struct{ MIME string; SampleRate, Channels, BitDepth int }
type Citation struct{ URI string; Start, End int; Title string }
type SafetyEvent struct{ Category, Action string; Score float32; Note string }

type Event struct {
  Type       EventType
  TextDelta  string
  AudioChunk []byte
  AudioFormat *AudioFormat
  Citations  []Citation
  Safety     *SafetyEvent
  ToolName   string
  ToolInput  json.RawMessage
  ToolResult any
  Raw        any
  Err        error
}

type TextStream interface {
  Events() <-chan Event
  Close() error
}

type Provider interface {
  GenerateText(ctx context.Context, req Request) (*TextResult, error)
  StreamText(ctx context.Context, req Request) (TextStream, error)
  GenerateObject[T any](ctx context.Context, req Request) (*ObjectResult[T], error)
  StreamObject[T any](ctx context.Context, req Request) (TextStream, error) // or ObjectStream[T] if we implement typed streaming
}
```

### `tools`

```go
type Meta struct{ CallID string; Messages []core.Message }
type Handle interface {
  Name() string
  InSchemaJSON() []byte
  OutSchemaJSON() []byte
  Exec(ctx context.Context, raw json.RawMessage, meta Meta) (any, error)
}

func New[I any, O any](name, desc string,
  exec func(context.Context, I, Meta) (O, error)) Handle
```

### `stream`

```go
func SSE(w http.ResponseWriter, s core.TextStream) error
func NDJSON(w http.ResponseWriter, s core.TextStream) error
```

### `obs`

```go
func Tracer() trace.Tracer
// internal helpers to start spans w/ standard attributes
```

---

## 7) Provider Adapters (Canonical Requirements)

For each adapter:

* **Transport**: Use shared `*http.Client`; set auth headers; timeouts; gzip; error mapping to `aierrors`.
* **Streaming**: Normalize provider-specific events to our `Event` sequence; handle chunk coalescing where needed; always send `EventFinish` or `EventError` exactly once.
* **Tools**: Parse provider tool call format → `ToolCall`; run with runner; re-inject results as messages; continue until `StopWhen`.
* **Structured outputs**: Prefer strict mode; else fallback to JSON + schema validation & repair.
* **Usage**: Extract when available; else let Phase 13 estimate.

**Adapters to implement:**

* OpenAI (canonical)
* Anthropic (SSE)
* Gemini (files, safety, citations)
* Ollama (local)
* OpenAI-compatible (Groq/xAI/Baseten/Cerebras) with quirks

---

## 8) Middleware (Retry/RateLimit/Safety/Caching)

* **Retry**: Wrap Provider; implement idempotency checks; default backoff strategy; count attempts in span attributes.
* **RateLimit**: Per-instance limiter; drop or queue with context cancel.
* **Safety**: Stream transformer that can redact or early-stop; synchronous final-text filter.

(An optional **caching** layer can be added later; out of scope for the initial release.)

---

## 9) Prompt System (Embedded + Overrides + Versions)

* Filenames carry version (`name@MAJOR.MINOR.PATCH.tmpl`).
* Registry loads from `embed.FS` plus optional override directory; override wins.
* Fingerprint each template; attach `{name, version, fingerprint}` to spans.
* CLI `prompts verify`: fails if content changed without version bump (used in CI).

---

## 10) Tools & JSON Schema Reflection

* Use `invopop/jsonschema` behind a light abstraction to convert Go types to JSON Schema once per type (cache).
* Respect `json:"name,omitempty"` tags for property names and required fields.
* Allow custom struct tags for enums / descriptions.

**Testing**: Golden schemas for a set of representative types to lock behavior.

---

## 11) Streaming (SSE/NDJSON) & Browser Integration

* SSE writer:

  * Headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`
  * Heartbeat every N seconds: `: keep-alive\n\n`
  * Write `data: {...}\n\n` for each event; flush (`Flusher`)
  * Close on context done / stream close / client disconnect

* NDJSON writer:

  * `Content-Type: application/x-ndjson`
  * One JSON object per line; flush periodically

**Testing**: `httptest` + simulated slow client; assert no goroutine leak.

---

## 12) Observability (OpenTelemetry) & Costs

* Start a span for each call; create child spans for each step & tool.
* Attributes documented and stable; test that they’re set.
* Optional cost estimation mapping by provider/model (table in code).

---

## 13) Evaluations & Sinks

* `Evaluator` interface; multiple evaluators can run per sample.
* Built-ins: exact, regex, contains, schema validate, LLM-as-judge.
* Sinks:

  * Braintrust: push runs & scores
  * Phoenix (Arize): trace and artifacts
  * LangSmith: trace spans & metadata

**Testing**: Mock HTTP; e2e CLI small dataset.

---

## 14) Audio (TTS/STT) & Built-in Speak Tool

* `SpeechProvider` with `Synthesize` returning a `SpeechStream`.
* `TranscriptionProvider` with `Transcribe`.
* Adapters: ElevenLabs (TTS), Cartesia (TTS), Whisper/Deepgram (STT).
* `Speak` tool that bridges LLM tool calls to TTS provider; returns a URL or file id.

**Testing**: Mock audio servers; generate small WAV files; ensure readable.

---

## 15) OpenAI-Compatible Adapter (Groq/xAI/Baseten/Cerebras)

* A single adapter with:

  * BaseURL & API key
  * Param filter (UnsupportedParams)
  * Flags: `DisableJSONStreaming`, `DisableParallelToolCalls`
  * Capability probe (`/v1/models`)

* Add **presets** for quick usage:

  * `presets.Groq()` etc.

**Testing**: Mock servers that reject unsupported params; assert trimming works.

---

## 16) Gemini-Specific Features (Files, Safety, Citations)

* File uploads: map `BlobRef{Bytes|URL}` → `file_data|inline_data`; store FileID in session.
* Safety config mapping; default thresholds.
* Stream citations: convert to `EventCitations`; attach to steps.

**Testing**: Mock server verifying multipart upload & citation frames.

---

## 17) MCP Support (Client & Server)

* **Client**: Connect (stdio/HTTP), `tools/list`, `tools/call`, `resources/list/read`, `prompts/list/get`.
  Wrap remote tools as dynamic `tools.Handle`; optional codegen `ai mcp codegen` to generate typed wrappers.

* **Server**: Expose our typed tools, resources, prompts. Support stdio & HTTP. Auth (Bearer/mTLS) for HTTP.

* **Policy**: allowlist of tools; time/size limits.

**Testing**: In-process client/server + subprocess stdio round-trip tests.

---

## 18) Memory & RAG Helpers

* Memory: in-memory ring buffer; SQLite persistent with compaction.
* RAG: sqlite-vec adapter; document chunking utility; embed pipeline hook.

**Testing**: Index, search, retrieval sanity for top‑k.

---

## 19) Router (Cost/Latency/A‑B/Failover)

* Policy expression → selector picks a provider/model:

  * Preflight token estimate
  * Context window check
  * Estimated cost/time (table or heuristics)

* AB: 0–1 proportion; sticky by request id hash if desired.

**Testing**: Deterministic selection under constraints; AB distribution within tolerance.

---

## 20) Token Estimation & Context Window Utilities

* Model metadata file: approximate tokens-per-character & context sizes.
* Estimator: count tokens per message; sum; approximate for non‑supported models.
* Split helper: chunk text to fit target window with overlap.

**Testing**: Fixtures for known counts; splitting edge cases.

---

## 21) Testing Strategy & Matrices

**Unit Tests**

* Aim 85%+ overall coverage, 90%+ in core/tools/prompts/stream.
* Fuzz tests:

  * JSON decode for tool inputs
  * Stream event parser/serializer
  * Prompt rendering with random inputs

**Integration Tests**

* Mock servers for each provider with golden fixtures for streams/tools/object outputs.
* Optional live tests (skipped by default) with API keys; safe & small payloads; separate job.

**End-to-End Tests**

* Example apps under `/examples` are compiled & run under CI:

  * SSE chat server contacting mock provider
  * Voice agent with mock TTS/STT
  * RAG demo with sqlite-vec
  * MCP import/export loopback

**Performance/Load**

* Benchmarks:

  * Runner throughput with tool calls (N tools, M steps)
  * SSE writer under throttled clients
  * Schema generation (cached vs cold)
* Budgets:

  * Steady-state allocations: O(1) per event where possible
  * Latency to first event under X ms in mock environment

**Static Analysis & Security**

* `staticcheck`, `gofumpt`,
* `govulncheck`, `gosec` (best-effort; suppress false positives with comments)

---

## 22) Security, Privacy & Compliance

* No PII logging by default; redact in telemetry (configurable).
* Tool sandboxing:

  * Allowlist tools and resource URI prefixes.
  * Max arg size (default 256 KB); max output size (default 2 MB).
  * Per-tool timeouts (default 30 s).
* HTTP: TLS by default; sensitive headers never logged.
* MCP HTTP: require Bearer token or mTLS; stdio servers inherit env and cwd (documented).
* Supply chain:

  * Dependabot/Renovate for deps
  * Checksum validation via Go modules
  * SBOM generation (optional in releases)

---

## 23) Docs, Examples, CLI & Website

* `/docs` Markdown + mkdocs/docusaurus config; publish via GitHub Pages.

* **Examples:**

  * `examples/hello-text`
  * `examples/hello-stream`
  * `examples/hello-object`
  * `examples/tools-weather`
  * `examples/web-sse`
  * `examples/voice-agent`
  * `examples/rag-sqlite-vec`
  * `examples/mcp-import-export`
  * `examples/router-ab`

* `cmd/ai` CLI:

  * `ai dev serve` (SSE server)
  * `ai eval run` (datasets → scores)
  * `ai prompts verify`
  * `ai mcp codegen` (optional)

---

## 24) CI/CD, Release Engineering & Versioning

* **CI jobs:** lint, unit, race, integration (mock), e2e (mock), live (optional gated).
* **Coverage gates**: 80% global; exceptions approved.
* **Releases:**

  * Tag via semver; generate changelog from PR titles/labels
  * `goreleaser` for CLI binaries (Linux/macOS/Windows)
  * Publish docs site on tag
* **Versioning & stability:**

  * Keep breaking API changes behind major versions (`v0` can be fast-evolving; stabilize before `v1.0.0`)
  * Deprecation policy: mark deprecated for one minor cycle before removing.

---

## 25) Contribution Guidelines & Governance

* **Issues & RFCs:** templates provided; design changes via lightweight RFC in `/docs/rfcs`.
* **Code review:** at least one owner approval; CI green required.
* **Style:** avoid exposed `any` types; prefer generics; public API must have examples and docs.
* **Community:** triage new providers as compatibility adapters first; add native adapters when unique features justify it.

---

# Implementation Nuggets & Scaffolds

To speed parallel work, here are a few ready-to-drop code fragments.

### `core/runner` step loop (skeleton)

```go
func runSteps(ctx context.Context, p Provider, req Request) (*TextResult, error) {
  // 1) Prepare messages (copy to avoid caller mutation)
  msgs := append([]Message(nil), req.Messages...)
  steps := make([]Step, 0, 4)

  for stepNum := 0; ; stepNum++ {
    select {
    case <-ctx.Done():
      return nil, ctx.Err()
    default:
    }

    // 2) Ask the provider for one step (non-streaming step API internally)
    stepRes, calls, usage, err := p.stepOnce(ctx, msgs, req) // adapter internal
    if err != nil { return nil, err }

    st := Step{Text: stepRes, ToolCalls: calls}
    if len(calls) == 0 {
      // No tools → final answer
      steps = append(steps, st)
      return &TextResult{
        Text: stepRes, Steps: steps, Usage: usage,
      }, nil
    }

    // 3) Execute tools in parallel
    results := make([]ToolExecution, len(calls))
    var wg sync.WaitGroup
    for i, c := range calls {
      wg.Add(1)
      go func(i int, c ToolCall) {
        defer wg.Done()
        h := findTool(req.Tools, c.Name)
        if h == nil { results[i] = ToolExecution{Name: c.Name, Result: map[string]any{"error":"unknown tool"}}; return }
        out, err := h.Exec(ctx, c.Input, Meta{CallID: fmt.Sprintf("%d", stepNum), Messages: msgs})
        if err != nil { results[i] = ToolExecution{Name: c.Name, Result: map[string]any{"error": err.Error()}}; return }
        results[i] = ToolExecution{Name: c.Name, Result: out}
      }(i, c)
    }
    wg.Wait()

    st.ToolResults = results
    steps = append(steps, st)

    // 4) Append tool results to messages
    msgs = append(msgs,
      Message{Role: Tool, Parts: []Part{Text{Text: encodeToolResults(results)}}},
    )

    // 5) StopWhen?
    if req.StopWhen != nil && req.StopWhen(len(steps), st) { break }
  }
  // If we break from the loop, finalize based on last step
  last := steps[len(steps)-1]
  return &TextResult{Text: last.Text, Steps: steps}, nil
}
```

### `stream/SSE` (skeleton)

```go
func SSE(w http.ResponseWriter, s core.TextStream) error {
  h := w.Header()
  h.Set("Content-Type", "text/event-stream")
  h.Set("Cache-Control", "no-cache")
  h.Set("Connection", "keep-alive")

  flusher, ok := w.(http.Flusher)
  if !ok { return errors.New("stream: flusher not supported") }

  ticker := time.NewTicker(15 * time.Second)
  defer ticker.Stop()

  enc := json.NewEncoder(w)

  for {
    select {
    case ev, ok := <-s.Events():
      if !ok {
        fmt.Fprint(w, "event: end\ndata: {}\n\n")
        flusher.Flush()
        return nil
      }
      fmt.Fprint(w, "data: ")
      if err := enc.Encode(ev); err != nil { return err }
      fmt.Fprint(w, "\n")
      flusher.Flush()
    case <-ticker.C:
      fmt.Fprint(w, ": keep-alive\n\n")
      flusher.Flush()
    }
  }
}
```

---

# Acceptance Gate: “Production-Ready” Checklist

Before `v0.9.0`:

* [ ] Core/Stream/Tools/Prompts/Obs: **complete & stable**
* [ ] Providers: **OpenAI, Anthropic, Gemini** production-ready; Ollama stable; OpenAI-compatible adapter stable w/ presets
* [ ] Middleware: retry/ratelimit/safety **done**
* [ ] SSE & NDJSON: **cross-browser tested**
* [ ] Audio: **ElevenLabs + one STT** adapter in “beta” (mocked tests; live optional)
* [ ] MCP: client + server **beta** (stdio + HTTP; policy defaults safe)
* [ ] Router & Token estimate: **beta**
* [ ] Memory & RAG (sqlite-vec): **beta**
* [ ] Evals + Braintrust sink: **beta**
* [ ] Examples: **all compile & run** with mock providers
* [ ] Docs site: **published**; godocs complete
* [ ] CI: unit (race), integration (mock), e2e (mock) **green**; live tests opt-in
* [ ] Security: gosec/govulncheck **clean**; secrets policy documented

Before `v1.0.0`:

* [ ] API review freeze; docstrings exhaustive
* [ ] Breaking changes queued for `v2` only
* [ ] Performance benchmarks published; budgets met
* [ ] Compatibility matrix documented (Go versions, OSes, providers)
* [ ] Contribution guide refined; governance clarified
* [ ] Signed release artifacts; SBOM (optional)

---

## How to Run Squads in Parallel (Summary)

* **Core/Stream/Tools/Prompts/Obs** (Phases 1–4) can run concurrently; each has minimal deps.
* **OpenAI provider (Phase 5)** can start once `core` and `tools` exist.
* **Middleware (6)** and **SSE/NDJSON (7)** in parallel to 5.
* **Anthropic (9)**, **Gemini (10)**, **Ollama (11)** can proceed in parallel after 5.
* **OpenAI-Compat (12)** depends on 5 but is largely parallel.
* **Audio (14)**, **Evals (15)**, **MCP (16)** proceed in parallel; only touch edges.
* **Memory/RAG (17)** and **Router (18)** parallel after token estimation (13).
* **Docs/Examples (19)** parallel from mid-project onward, continuously.

---

### Final Note

This plan yields a **Go-native, high‑performance, typed** AI framework with first-class streaming, structured outputs, tools, prompts, observability, evals, audio, MCP, and cross-provider support—including Gemini specialties and OpenAI-compatible ecosystems. The phases and tracks are deliberately **parallelizable** so multiple agents/teams can implement and test features concurrently while converging on a cohesive, production-quality library.
