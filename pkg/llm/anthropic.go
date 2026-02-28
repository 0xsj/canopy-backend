package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Verify anthropicProvider implements StreamProvider.
var _ StreamProvider = (*anthropicProvider)(nil)

// anthropicProvider adapts the Anthropic Messages API to the Provider port.
type anthropicProvider struct {
	client anthropic.Client
	cfg    Config
	log    logger.Logger
}

func newAnthropicProvider(cfg Config, log logger.Logger) *anthropicProvider {
	client := anthropic.NewClient(anthropicoption.WithAPIKey(cfg.APIKey))
	return &anthropicProvider{client: client, cfg: cfg, log: log}
}

func (p *anthropicProvider) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	const op = "llm: anthropic"

	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}

	system, messages := p.buildAnthropicMessages(req.Messages)

	maxTokens := int64(req.Options.MaxTokens)
	if maxTokens == 0 {
		maxTokens = int64(p.cfg.MaxTokens)
	}
	if maxTokens == 0 {
		maxTokens = 4096
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		Messages:  messages,
	}

	if len(system) > 0 {
		params.System = system
	}

	if req.Options.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Options.Temperature)
	}
	if req.Options.TopP > 0 {
		params.TopP = anthropic.Float(req.Options.TopP)
	}
	if len(req.Options.Stop) > 0 {
		params.StopSequences = req.Options.Stop
	}

	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	// Extract text content from response blocks.
	var content strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
	}

	cacheCreation := int(resp.Usage.CacheCreationInputTokens)
	cacheRead := int(resp.Usage.CacheReadInputTokens)

	p.log.Debug("anthropic call completed",
		logger.String("model", string(resp.Model)),
		logger.Int("input_tokens", int(resp.Usage.InputTokens)),
		logger.Int("output_tokens", int(resp.Usage.OutputTokens)),
		logger.Int("cache_creation_tokens", cacheCreation),
		logger.Int("cache_read_tokens", cacheRead),
	)

	return ChatResponse{
		Content: content.String(),
		Model:   string(resp.Model),
		Usage: Usage{
			InputTokens:         int(resp.Usage.InputTokens),
			OutputTokens:        int(resp.Usage.OutputTokens),
			CacheCreationTokens: cacheCreation,
			CacheReadTokens:     cacheRead,
		},
	}, nil
}

func (p *anthropicProvider) ChatCompletionStream(ctx context.Context, req ChatRequest, handler StreamHandler) error {
	const op = "llm: anthropic stream"

	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}

	system, messages := p.buildAnthropicMessages(req.Messages)

	maxTokens := int64(req.Options.MaxTokens)
	if maxTokens == 0 {
		maxTokens = int64(p.cfg.MaxTokens)
	}
	if maxTokens == 0 {
		maxTokens = 4096
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		Messages:  messages,
	}

	if len(system) > 0 {
		params.System = system
	}
	if req.Options.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Options.Temperature)
	}
	if req.Options.TopP > 0 {
		params.TopP = anthropic.Float(req.Options.TopP)
	}
	if len(req.Options.Stop) > 0 {
		params.StopSequences = req.Options.Stop
	}

	stream := p.client.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	var usage Usage

	for stream.Next() {
		event := stream.Current()

		switch event.Type {
		case "message_start":
			start := event.AsMessageStart()
			usage.InputTokens = int(start.Message.Usage.InputTokens)
			usage.CacheCreationTokens = int(start.Message.Usage.CacheCreationInputTokens)
			usage.CacheReadTokens = int(start.Message.Usage.CacheReadInputTokens)
		case "content_block_delta":
			delta := event.AsContentBlockDelta()
			if delta.Delta.Type == "text_delta" {
				td := delta.Delta.AsTextDelta()
				if err := handler(StreamChunk{Delta: td.Text}); err != nil {
					return fmt.Errorf("%s: handler: %w", op, err)
				}
			}
		case "message_delta":
			md := event.AsMessageDelta()
			usage.OutputTokens = int(md.Usage.OutputTokens)
		}
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	p.log.Debug("anthropic stream completed",
		logger.String("model", model),
		logger.Int("input_tokens", usage.InputTokens),
		logger.Int("output_tokens", usage.OutputTokens),
		logger.Int("cache_creation_tokens", usage.CacheCreationTokens),
		logger.Int("cache_read_tokens", usage.CacheReadTokens),
	)

	return handler(StreamChunk{Done: true, Usage: &usage})
}

// buildAnthropicMessages separates system messages and marks the last system
// block with a cache_control breakpoint for prompt caching. This tells Anthropic
// to cache the system prefix (session type prompt + seed/leaf context) so
// subsequent turns in the same session get faster TTFT and 90% cheaper input.
func (p *anthropicProvider) buildAnthropicMessages(msgs []Message) ([]anthropic.TextBlockParam, []anthropic.MessageParam) {
	var system []anthropic.TextBlockParam
	var messages []anthropic.MessageParam

	for _, msg := range msgs {
		switch msg.Role {
		case "system":
			system = append(system, anthropic.TextBlockParam{Text: msg.Content})
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}

	// Mark the last system block as the cache boundary. Everything up to and
	// including this block forms the stable prefix that Anthropic caches.
	if len(system) > 0 {
		system[len(system)-1].CacheControl = anthropic.NewCacheControlEphemeralParam()
	}

	return system, messages
}
