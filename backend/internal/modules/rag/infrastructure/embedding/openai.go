package embedding

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIEmbedder struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
	m := openai.LargeEmbedding3 // text-embedding-3-large (3072 dims)
	if model == "text-embedding-3-small" {
		m = openai.SmallEmbedding3 // 1536 dims — cheaper, fast
	}
	return &OpenAIEmbedder{client: openai.NewClient(apiKey), model: m}
}

func (e *OpenAIEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	vecs, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// EmbedBatch sends up to 2048 strings in a single API call.
func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequestStrings{
		Input: texts,
		Model: e.model,
	})
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}

	out := make([][]float32, len(resp.Data))
	for _, d := range resp.Data {
		out[d.Index] = d.Embedding
	}
	return out, nil
}
