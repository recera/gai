// Package media provides Text-to-Speech (TTS) and Speech-to-Text (STT) capabilities
// for the GAI framework, enabling voice-based AI interactions.
package media

import (
	"context"
	"io"
	"time"

	"github.com/recera/gai/core"
)

// SpeechProvider synthesizes text into speech audio.
type SpeechProvider interface {
	// Synthesize converts text to speech, returning a stream of audio chunks.
	Synthesize(ctx context.Context, req SpeechRequest) (SpeechStream, error)

	// ListVoices returns available voices for this provider.
	ListVoices(ctx context.Context) ([]Voice, error)
}

// TranscriptionProvider converts speech audio to text.
type TranscriptionProvider interface {
	// Transcribe converts audio to text.
	Transcribe(ctx context.Context, req TranscriptionRequest) (*TranscriptionResult, error)

	// TranscribeStream processes streaming audio input.
	TranscribeStream(ctx context.Context, audio io.Reader) (TranscriptionStream, error)
}

// SpeechRequest configures text-to-speech synthesis.
type SpeechRequest struct {
	// Text to synthesize (required).
	Text string

	// Voice ID or name (provider-specific).
	Voice string

	// Model to use (provider-specific, e.g., "eleven_multilingual_v2").
	Model string

	// Output format (e.g., "mp3", "pcm", "opus").
	Format string

	// Speaking speed (0.5 to 2.0, 1.0 is normal).
	Speed float32

	// Voice stability (provider-specific, 0.0 to 1.0).
	Stability float32

	// Voice similarity boost (provider-specific, 0.0 to 1.0).
	SimilarityBoost float32

	// Additional provider-specific options.
	Options map[string]any
}

// SpeechStream provides streaming audio output from TTS.
type SpeechStream interface {
	// Chunks returns a channel of audio data chunks.
	Chunks() <-chan []byte

	// Format returns the audio format information.
	Format() AudioFormat

	// Close stops the stream and releases resources.
	Close() error

	// Error returns any error that occurred during streaming.
	Error() error
}

// TranscriptionRequest configures speech-to-text transcription.
type TranscriptionRequest struct {
	// Audio input source.
	Audio core.BlobRef

	// Language code (e.g., "en", "es", "fr").
	Language string

	// Model to use (provider-specific).
	Model string

	// Enable punctuation restoration.
	Punctuate bool

	// Enable speaker diarization.
	Diarize bool

	// Enable profanity filtering.
	FilterProfanity bool

	// Custom vocabulary/keywords.
	Keywords []string

	// Maximum alternatives to return.
	MaxAlternatives int

	// Additional provider-specific options.
	Options map[string]any
}

// TranscriptionResult contains the transcribed text and metadata.
type TranscriptionResult struct {
	// Primary transcription text.
	Text string

	// Alternative transcriptions with confidence scores.
	Alternatives []TranscriptionAlternative

	// Word-level timing information.
	Words []WordTiming

	// Detected language (if auto-detected).
	Language string

	// Overall confidence score (0.0 to 1.0).
	Confidence float32

	// Duration of the audio.
	Duration time.Duration

	// Speaker segments (if diarization enabled).
	Speakers []SpeakerSegment
}

// TranscriptionAlternative represents an alternative transcription.
type TranscriptionAlternative struct {
	Text       string
	Confidence float32
}

// WordTiming provides timing information for individual words.
type WordTiming struct {
	Word       string
	Start      time.Duration
	End        time.Duration
	Confidence float32
	Speaker    int // Speaker ID if diarization enabled
}

// SpeakerSegment identifies a speaker's portion of the audio.
type SpeakerSegment struct {
	Speaker int
	Start   time.Duration
	End     time.Duration
	Text    string
}

// TranscriptionStream provides real-time transcription of streaming audio.
type TranscriptionStream interface {
	// Events returns a channel of transcription events.
	Events() <-chan TranscriptionEvent

	// Close stops the stream and releases resources.
	Close() error
}

// TranscriptionEvent represents a real-time transcription update.
type TranscriptionEvent struct {
	// Type of event.
	Type TranscriptionEventType

	// Transcribed text (for partial and final results).
	Text string

	// Whether this is a final result.
	IsFinal bool

	// Word timing (if available).
	Words []WordTiming

	// Error (for error events).
	Error error
}

// TranscriptionEventType identifies the type of transcription event.
type TranscriptionEventType int

const (
	TranscriptionPartial TranscriptionEventType = iota
	TranscriptionFinal
	TranscriptionError
	TranscriptionEnd
)

// AudioFormat describes audio encoding and properties.
type AudioFormat struct {
	// MIME type (e.g., "audio/mpeg", "audio/wav").
	MIME string

	// Sample rate in Hz (e.g., 44100, 16000).
	SampleRate int

	// Number of channels (1 for mono, 2 for stereo).
	Channels int

	// Bit depth (e.g., 16, 24).
	BitDepth int

	// Encoding format (e.g., "pcm", "mp3", "opus").
	Encoding string

	// Bitrate in bits per second (for compressed formats).
	Bitrate int
}

// Voice represents an available TTS voice.
type Voice struct {
	// Unique voice identifier.
	ID string

	// Human-readable voice name.
	Name string

	// Voice description.
	Description string

	// Language codes supported by this voice.
	Languages []string

	// Voice gender (if specified).
	Gender string

	// Voice age category (e.g., "young", "middle-aged", "old").
	Age string

	// Voice style/use case tags (e.g., "conversational", "narrative").
	Tags []string

	// Preview audio URL (if available).
	PreviewURL string

	// Whether this is a premium voice.
	Premium bool
}

// Common audio formats
const (
	FormatMP3    = "mp3"
	FormatWAV    = "wav"
	FormatOGG    = "ogg"
	FormatOpus   = "opus"
	FormatFLAC   = "flac"
	FormatPCM    = "pcm"
	FormatWebM   = "webm"
	FormatMPEG   = "mpeg"
	FormatULaw   = "ulaw"
	FormatMuLaw  = "mulaw"
)

// Common MIME types
const (
	MimeMP3   = "audio/mpeg"
	MimeWAV   = "audio/wav"
	MimeOGG   = "audio/ogg"
	MimeOpus  = "audio/opus"
	MimeFLAC  = "audio/flac"
	MimeWebM  = "audio/webm"
	MimeBasic = "audio/basic" // for ulaw
)

// ProviderConfig holds common configuration for audio providers.
type ProviderConfig struct {
	// API key for authentication.
	APIKey string

	// Base URL for the API.
	BaseURL string

	// Organization ID (if applicable).
	Organization string

	// Project ID (if applicable).
	Project string

	// Default voice to use.
	DefaultVoice string

	// Default model to use.
	DefaultModel string

	// Default audio format.
	DefaultFormat string

	// Request timeout.
	Timeout time.Duration

	// Maximum retries for failed requests.
	MaxRetries int

	// Custom HTTP headers.
	Headers map[string]string
}