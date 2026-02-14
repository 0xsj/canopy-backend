package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// geminiProvider adapts the Google Gemini API to the Provider port.
type geminiProvider struct {
	client *genai.Client
	cfg    Config
	log    logger.Logger
}

func newGeminiProvider(ctx context.Context, cfg Config, log logger.Logger) (*geminiProvider, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.APIKey))
	if err != nil {
		return nil, fmt.Errorf("llm: gemini: create client: %w", err)
	}
	return &geminiProvider{client: client, cfg: cfg, log: log}, nil
}

func (p *geminiProvider) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	const op = "llm: gemini"

	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}

	gm := p.client.GenerativeModel(model)

	// Extract system instruction from messages.
	var history []*genai.Content
	var lastUserParts []genai.Part

	for i, msg := range req.Messages {
		switch msg.Role {
		case "system":
			gm.SystemInstruction = genai.NewUserContent(genai.Text(msg.Content))
		case "user":
			if i == len(req.Messages)-1 {
				// Last user message is sent via SendMessage.
				lastUserParts = append(lastUserParts, genai.Text(msg.Content))
			} else {
				history = append(history, &genai.Content{
					Parts: []genai.Part{genai.Text(msg.Content)},
					Role:  "user",
				})
			}
		case "assistant":
			history = append(history, &genai.Content{
				Parts: []genai.Part{genai.Text(msg.Content)},
				Role:  "model",
			})
		}
	}

	// Generation config.
	genCfg := &genai.GenerationConfig{}
	if req.Options.Temperature > 0 {
		genCfg.SetTemperature(float32(req.Options.Temperature))
	} else if p.cfg.Temperature > 0 {
		genCfg.SetTemperature(float32(p.cfg.Temperature))
	}
	maxTokens := req.Options.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.cfg.MaxTokens
	}
	if maxTokens > 0 {
		genCfg.SetMaxOutputTokens(int32(maxTokens))
	}
	if req.Options.TopP > 0 {
		genCfg.SetTopP(float32(req.Options.TopP))
	}
	gm.GenerationConfig = *genCfg

	if len(req.Options.Stop) > 0 {
		gm.StopSequences = req.Options.Stop
	}

	// Use chat session for multi-turn.
	cs := gm.StartChat()
	cs.History = history

	if len(lastUserParts) == 0 {
		return ChatResponse{}, fmt.Errorf("%s: no user message to send", op)
	}

	resp, err := cs.SendMessage(ctx, lastUserParts...)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	if len(resp.Candidates) == 0 {
		return ChatResponse{}, fmt.Errorf("%s: no candidates in response", op)
	}

	cand := resp.Candidates[0]
	if cand.Content == nil || len(cand.Content.Parts) == 0 {
		return ChatResponse{}, fmt.Errorf("%s: empty content in response", op)
	}

	var content strings.Builder
	for _, part := range cand.Content.Parts {
		content.WriteString(fmt.Sprintf("%v", part))
	}

	var usage Usage
	if resp.UsageMetadata != nil {
		usage.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
		usage.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}

	p.log.Debug("gemini call completed",
		logger.String("model", model),
		logger.Int("input_tokens", usage.InputTokens),
		logger.Int("output_tokens", usage.OutputTokens),
	)

	return ChatResponse{
		Content: content.String(),
		Model:   model,
		Usage:   usage,
	}, nil
}
