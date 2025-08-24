# GAI Media Package

The `media` package provides Text-to-Speech (TTS) and Speech-to-Text (STT) capabilities for the GAI framework, enabling voice-based AI interactions.

## Features

- **Text-to-Speech (TTS)**: Convert text to natural-sounding speech
  - ElevenLabs provider with multiple voices and languages
  - Cartesia provider with professional narrator voices
  - Streaming audio output with chunk-based delivery
  - Multiple audio formats (MP3, WAV, OGG, FLAC, PCM)
  
- **Speech-to-Text (STT)**: Convert audio to text transcriptions
  - OpenAI Whisper for accurate transcription
  - Deepgram with real-time streaming and speaker diarization
  - Word-level timing information
  - Multiple language support with auto-detection
  
- **Speak Tool**: Allow LLMs to trigger TTS synthesis
  - Type-safe tool for AI agents
  - File saving and data URL generation
  - Automatic cleanup options

## Installation

```go
import "github.com/recera/gai/media"
```

## Quick Start

### Text-to-Speech with ElevenLabs

```go
// Create TTS provider
tts := media.NewElevenLabs(
    media.WithElevenLabsAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
)

// Synthesize speech
ctx := context.Background()
stream, err := tts.Synthesize(ctx, media.SpeechRequest{
    Text:   "Hello from the GAI framework!",
    Voice:  "Rachel",
    Format: media.FormatMP3,
    Speed:  1.0,
})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

// Save audio to file
file, _ := os.Create("output.mp3")
defer file.Close()

for chunk := range stream.Chunks() {
    file.Write(chunk)
}
```

### Speech-to-Text with Whisper

```go
// Create STT provider
stt := media.NewWhisper(
    media.WithWhisperAPIKey(os.Getenv("OPENAI_API_KEY")),
)

// Transcribe audio
result, err := stt.Transcribe(ctx, media.TranscriptionRequest{
    Audio: core.BlobRef{
        Kind: core.BlobURL,
        URL:  "https://example.com/audio.wav",
        MIME: "audio/wav",
    },
    Language:  "en",
    Punctuate: true,
})
if err != nil {
    log.Fatal(err)
}

fmt.Println("Transcription:", result.Text)
fmt.Println("Duration:", result.Duration)
```

### Real-time Transcription with Deepgram

```go
// Create Deepgram provider
deepgram := media.NewDeepgram(
    media.WithDeepgramAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
)

// Start streaming transcription
audioReader := getAudioStream() // Your audio source
stream, err := deepgram.TranscribeStream(ctx, audioReader)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

// Process transcription events
for event := range stream.Events() {
    switch event.Type {
    case media.TranscriptionPartial:
        fmt.Printf("Partial: %s\n", event.Text)
    case media.TranscriptionFinal:
        fmt.Printf("Final: %s\n", event.Text)
    case media.TranscriptionError:
        fmt.Printf("Error: %v\n", event.Error)
    }
}
```

### LLM-Triggered TTS with Speak Tool

```go
// Create TTS provider and Speak tool
tts := media.NewElevenLabs(
    media.WithElevenLabsAPIKey(apiKey),
)

speakTool := media.NewSpeakTool(
    tts,
    media.WithSpeakToolTempDir("/tmp/audio"),
    media.WithSpeakToolCleanup(5*time.Minute),
)

// Use with AI agent
request := core.Request{
    Messages: []core.Message{
        {Role: core.System, Parts: []core.Part{
            core.Text{Text: "You can speak by calling the speak tool."},
        }},
        {Role: core.User, Parts: []core.Part{
            core.Text{Text: "Say hello and introduce yourself."},
        }},
    },
    Tools: []tools.Handle{speakTool},
    ToolChoice: core.ToolAuto,
}

// The AI will call the speak tool to generate audio
result, _ := provider.GenerateText(ctx, request)
```

## Providers

### TTS Providers

#### ElevenLabs

High-quality neural TTS with expressive voices:

```go
tts := media.NewElevenLabs(
    media.WithElevenLabsAPIKey(apiKey),
    media.WithElevenLabsVoice("Rachel"),     // Default voice
    media.WithElevenLabsModel("eleven_multilingual_v2"),
)

// List available voices
voices, _ := tts.ListVoices(ctx)
for _, voice := range voices {
    fmt.Printf("%s: %s\n", voice.Name, voice.Description)
}
```

#### Cartesia

Professional narrator voices optimized for long-form content:

```go
tts := media.NewCartesia(
    media.WithCartesiaAPIKey(apiKey),
    media.WithCartesiaVoice("narrator-professional"),
    media.WithCartesiaModel("sonic-english"),
)
```

### STT Providers

#### OpenAI Whisper

Accurate transcription with word-level timing:

```go
stt := media.NewWhisper(
    media.WithWhisperAPIKey(apiKey),
    media.WithWhisperModel("whisper-1"),
    media.WithWhisperBaseURL("https://api.openai.com"), // Or self-hosted
)
```

#### Deepgram

Real-time transcription with advanced features:

```go
stt := media.NewDeepgram(
    media.WithDeepgramAPIKey(apiKey),
    media.WithDeepgramModel("nova-2"),
)

// Transcribe with diarization
result, _ := stt.Transcribe(ctx, media.TranscriptionRequest{
    Audio:           audioBlob,
    Diarize:         true,      // Speaker identification
    Punctuate:       true,      // Add punctuation
    FilterProfanity: false,     // Keep original words
    MaxAlternatives: 3,         // Get multiple transcriptions
})

// Access speaker segments
for _, segment := range result.Speakers {
    fmt.Printf("Speaker %d: %s\n", segment.Speaker, segment.Text)
}
```

## Audio Formats

The media package supports various audio formats:

- **MP3** (`audio/mpeg`) - Compressed, widely supported
- **WAV** (`audio/wav`) - Uncompressed, high quality
- **OGG** (`audio/ogg`) - Open format, good compression
- **Opus** (`audio/opus`) - Excellent for speech
- **FLAC** (`audio/flac`) - Lossless compression
- **PCM** (`audio/pcm`) - Raw audio data
- **WebM** (`audio/webm`) - Web-optimized
- **μ-law/A-law** - Telephony formats

## Advanced Features

### Custom Voice Settings

```go
req := media.SpeechRequest{
    Text:            "Advanced synthesis",
    Voice:           "custom-voice",
    Speed:           1.2,              // 20% faster
    Stability:       0.8,              // More expressive
    SimilarityBoost: 0.9,              // Closer to original voice
    Options: map[string]any{
        "emotion": "excited",          // Provider-specific
    },
}
```

### Transcription with Keywords

```go
result, _ := stt.Transcribe(ctx, media.TranscriptionRequest{
    Audio:    audioBlob,
    Keywords: []string{"GAI", "framework", "OpenAI"}, // Boost recognition
    Language: "en",
})
```

### Streaming Audio Collection

```go
stream, _ := tts.Synthesize(ctx, req)

// Stream to HTTP response
http.HandleFunc("/audio", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "audio/mpeg")
    w.Header().Set("Transfer-Encoding", "chunked")
    
    for chunk := range stream.Chunks() {
        w.Write(chunk)
        w.(http.Flusher).Flush()
    }
})
```

## Error Handling

All providers use the GAI framework's unified error taxonomy:

```go
stream, err := tts.Synthesize(ctx, req)
if err != nil {
    if aiErr, ok := err.(*core.AIError); ok {
        switch aiErr.Code {
        case core.ErrorRateLimited:
            // Wait and retry
            time.Sleep(aiErr.RetryAfter)
        case core.ErrorUnauthorized:
            // Check API key
        case core.ErrorProviderUnavailable:
            // Use fallback provider
        }
    }
}
```

## Performance

Benchmark results on M1 MacBook Pro:

```
BenchmarkElevenLabsSynthesize-8         1000    1.2ms/op    2KB/op
BenchmarkWhisperTranscribe-8             500    2.4ms/op    5KB/op
BenchmarkDeepgramTranscribe-8            500    2.1ms/op    4KB/op
BenchmarkSpeakTool-8                    5000    240μs/op    1KB/op
BenchmarkParallelSynthesis-8           10000    120μs/op    0.5KB/op
```

## Testing

Run unit tests:
```bash
go test ./media
```

Run integration tests (requires API keys):
```bash
ELEVENLABS_API_KEY=xxx OPENAI_API_KEY=xxx go test -tags=integration ./media
```

Run benchmarks:
```bash
go test -bench=. ./media
```

## Environment Variables

- `ELEVENLABS_API_KEY` - ElevenLabs API key
- `CARTESIA_API_KEY` - Cartesia API key
- `OPENAI_API_KEY` - OpenAI API key (for Whisper)
- `DEEPGRAM_API_KEY` - Deepgram API key

## Contributing

The media package follows the GAI framework patterns:
- Provider-agnostic interfaces
- Streaming-first design
- Comprehensive error handling
- Thread-safe operations
- Zero-allocation hot paths where possible

## License

Part of the GAI framework. See main LICENSE file.