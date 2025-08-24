package media_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/media"
	"github.com/recera/gai/providers/openai"
	"github.com/recera/gai/tools"
)

// Example demonstrates basic TTS synthesis with ElevenLabs
func ExampleElevenLabs_Synthesize() {
	// Create TTS provider
	tts := media.NewElevenLabs(
		media.WithElevenLabsAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
	)

	// Synthesize speech
	ctx := context.Background()
	stream, err := tts.Synthesize(ctx, media.SpeechRequest{
		Text:   "Hello from the GAI framework! This is a test of text-to-speech synthesis.",
		Voice:  "Rachel",
		Format: media.FormatMP3,
		Speed:  1.0,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	// Collect audio data
	var totalBytes int
	for chunk := range stream.Chunks() {
		totalBytes += len(chunk)
		// In real use, write to file or stream to client
	}

	fmt.Printf("Generated %d bytes of MP3 audio\n", totalBytes)
}

// Example demonstrates listing available voices
func ExampleElevenLabs_ListVoices() {
	tts := media.NewElevenLabs(
		media.WithElevenLabsAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
	)

	ctx := context.Background()
	voices, err := tts.ListVoices(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Display first 3 voices
	for i, voice := range voices {
		if i >= 3 {
			break
		}
		fmt.Printf("Voice: %s\n", voice.Name)
		fmt.Printf("  ID: %s\n", voice.ID)
		fmt.Printf("  Gender: %s, Age: %s\n", voice.Gender, voice.Age)
		fmt.Printf("  Premium: %v\n", voice.Premium)
		fmt.Println()
	}
}

// Example demonstrates transcribing audio with Whisper
func ExampleWhisper_Transcribe() {
	// Create STT provider
	stt := media.NewWhisper(
		media.WithWhisperAPIKey(os.Getenv("OPENAI_API_KEY")),
	)

	// Transcribe audio from URL
	ctx := context.Background()
	result, err := stt.Transcribe(ctx, media.TranscriptionRequest{
		Audio: core.BlobRef{
			Kind: core.BlobURL,
			URL:  "https://example.com/sample-audio.wav",
			MIME: "audio/wav",
		},
		Language:  "en",
		Punctuate: true,
		Keywords:  []string{"GAI", "framework", "artificial intelligence"},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcription: %s\n", result.Text)
	fmt.Printf("Language: %s\n", result.Language)
	fmt.Printf("Duration: %v\n", result.Duration)

	// Show word timings if available
	if len(result.Words) > 0 {
		fmt.Println("\nFirst 5 words with timing:")
		for i, word := range result.Words {
			if i >= 5 {
				break
			}
			fmt.Printf("  %s: %v-%v\n", word.Word, word.Start, word.End)
		}
	}
}

// Example demonstrates real-time transcription with Deepgram
func ExampleDeepgram_Transcribe() {
	// Create Deepgram provider
	deepgram := media.NewDeepgram(
		media.WithDeepgramAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
		media.WithDeepgramModel("nova-2"),
	)

	// Transcribe with advanced features
	ctx := context.Background()
	result, err := deepgram.Transcribe(ctx, media.TranscriptionRequest{
		Audio: core.BlobRef{
			Kind:  core.BlobBytes,
			Bytes: getAudioBytes(), // Your audio data
			MIME:  "audio/wav",
		},
		Language:        "en",
		Punctuate:       true,
		Diarize:         true, // Enable speaker identification
		FilterProfanity: false,
		MaxAlternatives: 2,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transcription: %s\n", result.Text)
	fmt.Printf("Confidence: %.2f\n", result.Confidence)

	// Show speaker segments
	if len(result.Speakers) > 0 {
		fmt.Println("\nSpeaker segments:")
		for _, segment := range result.Speakers {
			fmt.Printf("  Speaker %d (%v-%v): %s\n",
				segment.Speaker, segment.Start, segment.End, segment.Text)
		}
	}

	// Show alternatives
	if len(result.Alternatives) > 0 {
		fmt.Println("\nAlternative transcriptions:")
		for i, alt := range result.Alternatives {
			fmt.Printf("  %d. (confidence %.2f): %s\n",
				i+1, alt.Confidence, alt.Text)
		}
	}
}

// Example demonstrates the Speak tool for LLM-triggered TTS
func ExampleNewSpeakTool() {
	// Create TTS provider
	tts := media.NewElevenLabs(
		media.WithElevenLabsAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
	)

	// Create Speak tool
	speakTool := media.NewSpeakTool(
		tts,
		media.WithSpeakToolTempDir("/tmp/audio"),
		media.WithSpeakToolMaxTextLength(1000),
		media.WithSpeakToolDefaultFormat(media.FormatMP3),
		media.WithSpeakToolCleanup(5*time.Minute),
	)

	// Create AI provider
	ai := openai.New(
		openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel("gpt-4o-mini"),
	)

	// Use the speak tool in an AI request
	ctx := context.Background()
	result, err := ai.GenerateText(ctx, core.Request{
		Messages: []core.Message{
			{Role: core.System, Parts: []core.Part{
				core.Text{Text: "You are a helpful assistant. When asked to speak or say something aloud, use the speak tool."},
			}},
			{Role: core.User, Parts: []core.Part{
				core.Text{Text: "Please say 'Welcome to the GAI framework' out loud."},
			}},
		},
		Tools:      []core.ToolHandle{tools.NewCoreAdapter(speakTool)},
		ToolChoice: core.ToolAuto,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("AI Response: %s\n", result.Text)

	// The AI will have called the speak tool
	// Check the steps for tool execution results
	for _, step := range result.Steps {
		for _, execution := range step.ToolResults {
			if execution.Name == "speak" {
				// The result contains the audio file path or data URL
				fmt.Printf("Audio generated: %+v\n", execution.Result)
			}
		}
	}
}

// Example demonstrates voice cloning workflow
func Example_voiceCloning() {
	// This example shows a complete voice cloning workflow:
	// 1. Transcribe reference audio
	// 2. Generate new speech in the same style
	// 3. Verify the output

	ctx := context.Background()

	// Step 1: Transcribe reference audio to get the text
	whisper := media.NewWhisper(
		media.WithWhisperAPIKey(os.Getenv("OPENAI_API_KEY")),
	)

	referenceAudio := core.BlobRef{
		Kind: core.BlobURL,
		URL:  "https://example.com/reference-voice.wav",
		MIME: "audio/wav",
	}

	transcription, err := whisper.Transcribe(ctx, media.TranscriptionRequest{
		Audio:    referenceAudio,
		Language: "en",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Reference text: %s\n", transcription.Text)

	// Step 2: Generate new speech with similar voice characteristics
	// (Note: This would require a voice cloning service or custom voice ID)
	elevenlabs := media.NewElevenLabs(
		media.WithElevenLabsAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
	)

	newText := "This is new text spoken in a similar voice style."
	stream, err := elevenlabs.Synthesize(ctx, media.SpeechRequest{
		Text:            newText,
		Voice:           "custom-cloned-voice", // Would be your cloned voice ID
		Format:          media.FormatMP3,
		Stability:       0.75, // Adjust for voice consistency
		SimilarityBoost: 0.90, // High similarity to original
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	// Collect synthesized audio
	var audioData []byte
	for chunk := range stream.Chunks() {
		audioData = append(audioData, chunk...)
	}

	// Step 3: Optionally transcribe the generated audio to verify
	verification, err := whisper.Transcribe(ctx, media.TranscriptionRequest{
		Audio: core.BlobRef{
			Kind:  core.BlobBytes,
			Bytes: audioData,
			MIME:  "audio/mpeg",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Generated text verification: %s\n", verification.Text)
	fmt.Printf("Match: %v\n", strings.Contains(verification.Text, "similar voice"))
}

// Example demonstrates audio format conversion
func Example_audioFormatConversion() {
	tts := media.NewCartesia(
		media.WithCartesiaAPIKey(os.Getenv("CARTESIA_API_KEY")),
	)

	ctx := context.Background()
	formats := []string{
		media.FormatMP3,
		media.FormatWAV,
		media.FormatOGG,
		media.FormatFLAC,
	}

	text := "Testing audio format conversion."

	for _, format := range formats {
		stream, err := tts.Synthesize(ctx, media.SpeechRequest{
			Text:   text,
			Format: format,
		})
		if err != nil {
			log.Printf("Failed to synthesize %s: %v", format, err)
			continue
		}

		// Get format info
		audioFormat := stream.Format()
		fmt.Printf("Format %s:\n", format)
		fmt.Printf("  MIME: %s\n", audioFormat.MIME)
		fmt.Printf("  Encoding: %s\n", audioFormat.Encoding)
		fmt.Printf("  Sample Rate: %d Hz\n", audioFormat.SampleRate)
		fmt.Printf("  Channels: %d\n", audioFormat.Channels)
		fmt.Printf("  Bit Depth: %d\n", audioFormat.BitDepth)

		stream.Close()
	}
}

// Example demonstrates creating an audio data URL for web usage
func Example_speakToolDataURL() {
	// Create mock provider for demonstration
	mockTTS := &mockProvider{}

	// Create speak tool configured for data URLs
	speakTool := media.NewSpeakTool(
		mockTTS,
		media.WithSpeakToolDefaultFormat(media.FormatMP3),
	)

	ctx := context.Background()
	meta := tools.Meta{
		CallID:    "example",
		RequestID: "req-123",
	}

	// Execute speak tool
	input := media.SpeakInput{
		Text:          "Hello, world!",
		Format:        media.FormatMP3,
		ReturnDataURL: true,
	}

	// Execute through the tool interface
	result, err := executeTool(speakTool, ctx, input, meta)
	if err != nil {
		log.Fatal(err)
	}

	output := result.(media.SpeakOutput)
	if output.Success {
		// Extract the base64 data
		parts := strings.Split(output.DataURL, ",")
		if len(parts) == 2 {
			decoded, _ := base64.StdEncoding.DecodeString(parts[1])
			fmt.Printf("Generated audio data URL with %d bytes\n", len(decoded))
			fmt.Printf("Can be used in HTML: <audio src='%s...' />\n",
				output.DataURL[:50])
		}
	}
}

// Helper functions for examples

func getAudioBytes() []byte {
	// In a real application, this would load actual audio data
	return []byte("fake audio data for example")
}

type mockProvider struct{}

func (m *mockProvider) Synthesize(ctx context.Context, req media.SpeechRequest) (media.SpeechStream, error) {
	return &mockStream{
		data: []byte("mock audio data"),
	}, nil
}

func (m *mockProvider) ListVoices(ctx context.Context) ([]media.Voice, error) {
	return []media.Voice{}, nil
}

type mockStream struct {
	data []byte
}

func (s *mockStream) Chunks() <-chan []byte {
	ch := make(chan []byte, 1)
	ch <- s.data
	close(ch)
	return ch
}

func (s *mockStream) Format() media.AudioFormat {
	return media.AudioFormat{
		MIME:     media.MimeMP3,
		Encoding: media.FormatMP3,
	}
}

func (s *mockStream) Close() error {
	return nil
}

func (s *mockStream) Error() error {
	return nil
}

func executeTool(tool tools.Handle, ctx context.Context, input media.SpeakInput, meta tools.Meta) (interface{}, error) {
	// This would normally be handled by the framework
	// Simplified for example purposes
	return media.SpeakOutput{
		Success:       true,
		DataURL:       "data:audio/mpeg;base64,bW9jayBhdWRpbyBkYXRh",
		Format:        media.FormatMP3,
		SizeBytes:     15,
		DurationSeconds: 1.0,
	}, nil
}