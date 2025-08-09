# Streaming & UI (SSE)

GAI streams `StreamChunk` structs in server code. A small adapter writes SSE compatible with modern UI hooks.

## Server

```go
http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
  ch := make(chan gai.StreamChunk)
  go func(){
    defer close(ch)
    _ = client.StreamCompletion(ctx, parts.Value(), func(s gai.StreamChunk) error {
      ch <- s
      return nil
    })
  }()
  uistream.Write(w, ch)
})
```
Notes:
- The server sets `Content-Type: text/event-stream` and `x-vercel-ai-ui-message-stream: v1` via `uistream.Write`.
- End events may include `Usage` when supported by the provider (enable for OpenAI with `WithOpenAIIncludeUsageInStream(true)`).

## Client

A React `useChat` example (from Vercel AI SDK):

```tsx
import { useChat } from ai/react

export default function Chat() {
  const { messages, input, setInput, handleSubmit } = useChat({ api: /api/chat })
  return (
    <form onSubmit={handleSubmit}>
      <ul>{messages.map(m => <li key={m.id}>{m.role}: {m.content}</li>)}</ul>
      <input value={input} onChange={e=>setInput(e.target.value)} />
    </form>
  )
}
```

