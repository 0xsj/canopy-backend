package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	openaioption "github.com/openai/openai-go/option"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Verify openaiProvider implements StreamProvider.
var _ StreamProvider = (*openaiProvider)(nil)

// openaiProvider adapts the OpenAI Chat Completions API to the Provider port.
type openaiProvider struct {
	client openai.Client
	cfg    Config
	log    logger.Logger
}

func newOpenAIProvider(cfg Config, log logger.Logger) *openaiProvider {
	client := openai.NewClient(openaioption.WithAPIKey(cfg.APIKey))
	return &openaiProvider{client: client, cfg: cfg, log: log}
}

func (p *openaiProvider) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	const op = "llm: openai"

	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}

	var messages []openai.ChatCompletionMessageParamUnion
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(model),
		Messages: messages,
	}

	if req.Options.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(req.Options.MaxTokens))
	} else if p.cfg.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(p.cfg.MaxTokens))
	}

	if req.Options.Temperature > 0 {
		params.Temperature = openai.Float(req.Options.Temperature)
	}
	if req.Options.TopP > 0 {
		params.TopP = openai.Float(req.Options.TopP)
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	if len(resp.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("%s: no choices in response", op)
	}

	content := resp.Choices[0].Message.Content

	p.log.Debug("openai call completed",
		logger.String("model", resp.Model),
		logger.Int("input_tokens", int(resp.Usage.PromptTokens)),
		logger.Int("output_tokens", int(resp.Usage.CompletionTokens)),
	)

	return ChatResponse{
		Content: content,
		Model:   resp.Model,
		Usage: Usage{
			InputTokens:  int(resp.Usage.PromptTokens),
			OutputTokens: int(resp.Usage.CompletionTokens),
		},
	}, nil
}

func (p *openaiProvider) ChatCompletionStream(ctx context.Context, req ChatRequest, handler StreamHandler) error {
	const op = "llm: openai stream"

	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}

	var messages []openai.ChatCompletionMessageParamUnion
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(model),
		Messages: messages,
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		},
	}

	if req.Options.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(req.Options.MaxTokens))
	} else if p.cfg.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(p.cfg.MaxTokens))
	}
	if req.Options.Temperature > 0 {
		params.Temperature = openai.Float(req.Options.Temperature)
	}
	if req.Options.TopP > 0 {
		params.TopP = openai.Float(req.Options.TopP)
	}

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)
	defer stream.Close()

	var usage Usage

	for stream.Next() {
		chunk := stream.Current()

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				if err := handler(StreamChunk{Delta: delta}); err != nil {
					return fmt.Errorf("%s: handler: %w", op, err)
				}
			}
		}

		if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
			usage.InputTokens = int(chunk.Usage.PromptTokens)
			usage.OutputTokens = int(chunk.Usage.CompletionTokens)
		}
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	p.log.Debug("openai stream completed",
		logger.String("model", model),
		logger.Int("input_tokens", usage.InputTokens),
		logger.Int("output_tokens", usage.OutputTokens),
	)

	return handler(StreamChunk{Done: true, Usage: &usage})
}
