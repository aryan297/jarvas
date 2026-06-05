package transcription

import (
	"bytes"
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type WhisperTranscriber struct {
	client *openai.Client
}

func NewWhisperTranscriber(apiKey string) *WhisperTranscriber {
	return &WhisperTranscriber{client: openai.NewClient(apiKey)}
}

// Transcribe sends audio bytes to OpenAI Whisper and returns the transcript text.
// filename must include the extension so Whisper knows the format (e.g. "audio.webm").
// language is an optional BCP-47 language code; empty string = auto-detect.
func (w *WhisperTranscriber) Transcribe(ctx context.Context, audioData []byte, filename, language string) (string, error) {
	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		Reader:   bytes.NewReader(audioData),
		FilePath: filename,
	}
	if language != "" {
		req.Language = language
	}

	resp, err := w.client.CreateTranscription(ctx, req)
	if err != nil {
		return "", fmt.Errorf("whisper transcription: %w", err)
	}
	return resp.Text, nil
}
