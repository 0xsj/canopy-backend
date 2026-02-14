package llm

import (
	"context"
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// NewProvider creates an LLM provider for the configured backend.
// Routes to the correct adapter based on cfg.Provider.
func NewProvider(ctx context.Context, cfg Config, log logger.Logger) (Provider, error) {
	switch cfg.Provider {
	case "anthropic":
		log.Info("llm provider initialized", logger.String("provider", "anthropic"), logger.String("model", cfg.Model))
		return newAnthropicProvider(cfg, log), nil

	case "openai":
		log.Info("llm provider initialized", logger.String("provider", "openai"), logger.String("model", cfg.Model))
		return newOpenAIProvider(cfg, log), nil

	case "gemini":
		p, err := newGeminiProvider(ctx, cfg, log)
		if err != nil {
			return nil, err
		}
		log.Info("llm provider initialized", logger.String("provider", "gemini"), logger.String("model", cfg.Model))
		return p, nil

	default:
		return nil, fmt.Errorf("llm: unknown provider %q", cfg.Provider)
	}
}
