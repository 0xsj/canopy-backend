package llm

import (
	"time"

	"github.com/0xsj/canopy-backend/pkg/config"
)

// Config holds LLM provider settings.
// Implements config.Section for use with the config loader.
type Config struct {
	Provider       string        // "anthropic", "openai", "gemini"
	Model          string        // e.g. "claude-sonnet-4-5-20250929"
	APIKey         string        // API key for the selected provider
	RequestTimeout time.Duration // per-call timeout
	Temperature    float64       // default temperature
	MaxTokens      int           // default max tokens
}

func (c *Config) Load(env config.EnvReader) {
	c.Provider = env.String("PROVIDER", "")
	c.Model = env.String("MODEL", "")
	c.APIKey = env.String("API_KEY", "")
	c.RequestTimeout = env.Duration("REQUEST_TIMEOUT", 60*time.Second)
	c.Temperature = env.Float64("TEMPERATURE", 0.7)
	c.MaxTokens = env.Int("MAX_TOKENS", 4096)
}

func (c *Config) Validate() error {
	var v config.Errors
	v.Required("provider", c.Provider)
	v.Required("model", c.Model)
	v.Required("api_key", c.APIKey)
	v.OneOf("provider", c.Provider, []string{"anthropic", "openai", "gemini"})
	v.Positive("max_tokens", c.MaxTokens)
	return v.Err()
}
