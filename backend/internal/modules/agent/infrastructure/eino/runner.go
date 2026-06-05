package eino

import (
	"context"
	"fmt"
	"strings"
	"time"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"

	chatenity "github.com/jarvas/backend/internal/modules/chat/domain/entity"
	toolsvc "github.com/jarvas/backend/internal/modules/tool/application/service"
)

const maxToolIterations = 5

// RunConfig holds everything needed for a single agent turn.
type RunConfig struct {
	OpenAIKey    string
	Model        string
	Temperature  float64
	MaxTokens    int
	SystemPrompt string
	History      []chatenity.Message
	UserMsg      string
	Tools        []*schema.ToolInfo
	Registry     *toolsvc.Registry
}

// Runner wraps Eino ChatModel with a tool-calling loop.
type Runner struct{}

func NewRunner() *Runner { return &Runner{} }

// Run executes a full agent turn and returns the final assistant text.
func (r *Runner) Run(ctx context.Context, cfg RunConfig) (string, error) {
	cm, err := buildModel(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("eino runner: build model: %w", err)
	}

	msgs := buildMessages(cfg.SystemPrompt, cfg.History, cfg.UserMsg)

	for i := 0; i < maxToolIterations; i++ {
		resp, err := cm.Generate(ctx, msgs)
		if err != nil {
			return "", fmt.Errorf("eino runner: generate: %w", err)
		}

		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		// Append assistant message with the tool calls.
		msgs = append(msgs, resp)

		// Execute each tool call and append results.
		for _, tc := range resp.ToolCalls {
			result, err := cfg.Registry.Execute(tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf("tool error: %s", err.Error())
			}
			msgs = append(msgs, schema.ToolMessage(result, tc.ID))
		}
	}

	// Safety: if we hit max iterations, ask the model to summarise.
	resp, err := cm.Generate(ctx, msgs)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// Stream executes the tool calling loop then emits the final response token-by-token.
func (r *Runner) Stream(ctx context.Context, cfg RunConfig) (<-chan string, error) {
	// Run the tool loop first (non-streaming for tool rounds).
	cm, err := buildModel(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("eino runner stream: build model: %w", err)
	}

	msgs := buildMessages(cfg.SystemPrompt, cfg.History, cfg.UserMsg)

	// Tool calling rounds — non-streaming.
	for i := 0; i < maxToolIterations; i++ {
		resp, err := cm.Generate(ctx, msgs)
		if err != nil {
			return nil, fmt.Errorf("eino runner stream: generate: %w", err)
		}
		if len(resp.ToolCalls) == 0 {
			// No more tool calls — stream this final content.
			ch := streamString(resp.Content)
			return ch, nil
		}
		msgs = append(msgs, resp)
		for _, tc := range resp.ToolCalls {
			result, execErr := cfg.Registry.Execute(tc.Function.Name, tc.Function.Arguments)
			if execErr != nil {
				result = fmt.Sprintf("tool error: %s", execErr.Error())
			}
			msgs = append(msgs, schema.ToolMessage(result, tc.ID))
		}
	}

	// Fallback: stream a final generate call.
	streamReader, err := cm.Stream(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("eino runner stream: stream: %w", err)
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		defer streamReader.Close()
		for {
			msg, err := streamReader.Recv()
			if err != nil {
				break
			}
			if msg.Content != "" {
				ch <- msg.Content
			}
		}
	}()
	return ch, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildModel(ctx context.Context, cfg RunConfig) (*openaimodel.ChatModel, error) {
	temp := float32(cfg.Temperature)
	maxTok := cfg.MaxTokens

	cm, err := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:      cfg.OpenAIKey,
		Model:       cfg.Model,
		MaxTokens:   &maxTok,
		Temperature: &temp,
	})
	if err != nil {
		return nil, err
	}

	if len(cfg.Tools) > 0 {
		if err := cm.BindTools(cfg.Tools); err != nil {
			return nil, fmt.Errorf("bind tools: %w", err)
		}
	}
	return cm, nil
}

func buildMessages(system string, history []chatenity.Message, userMsg string) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(history)+2)
	msgs = append(msgs, schema.SystemMessage(system))
	for _, h := range history {
		if h.Role == chatenity.RoleUser {
			msgs = append(msgs, schema.UserMessage(h.Content))
		} else if h.Role == chatenity.RoleAssistant {
			msgs = append(msgs, schema.AssistantMessage(h.Content, nil))
		}
	}
	msgs = append(msgs, schema.UserMessage(userMsg))
	return msgs
}

// streamString emits a pre-built string as word-by-word chunks over a channel.
func streamString(s string) <-chan string {
	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		words := strings.Fields(s)
		for i, w := range words {
			chunk := w
			if i < len(words)-1 {
				chunk += " "
			}
			ch <- chunk
			time.Sleep(8 * time.Millisecond)
		}
	}()
	return ch
}
