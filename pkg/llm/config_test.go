package llm

import (
	"testing"
	"time"

	"github.com/0xsj/canopy-backend/pkg/config"
)

func TestConfig_Load_Defaults(t *testing.T) {
	var cfg Config
	env := config.NewEnvReader("TEST_LLM")
	cfg.Load(env)

	if cfg.Provider != "" {
		t.Errorf("Provider = %q, want empty default", cfg.Provider)
	}
	if cfg.Model != "" {
		t.Errorf("Model = %q, want empty default", cfg.Model)
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty default", cfg.APIKey)
	}
	if cfg.RequestTimeout != 60*time.Second {
		t.Errorf("RequestTimeout = %v, want 60s", cfg.RequestTimeout)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", cfg.Temperature)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096", cfg.MaxTokens)
	}
}

func TestConfig_Load_FromEnv(t *testing.T) {
	t.Setenv("TEST_LLM_PROVIDER", "anthropic")
	t.Setenv("TEST_LLM_MODEL", "claude-sonnet-4-5-20250929")
	t.Setenv("TEST_LLM_API_KEY", "sk-ant-test")
	t.Setenv("TEST_LLM_REQUEST_TIMEOUT", "30s")
	t.Setenv("TEST_LLM_TEMPERATURE", "0.5")
	t.Setenv("TEST_LLM_MAX_TOKENS", "8192")

	var cfg Config
	env := config.NewEnvReader("TEST_LLM")
	cfg.Load(env)

	if cfg.Provider != "anthropic" {
		t.Errorf("Provider = %q, want anthropic", cfg.Provider)
	}
	if cfg.Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("Model = %q, want claude-sonnet-4-5-20250929", cfg.Model)
	}
	if cfg.APIKey != "sk-ant-test" {
		t.Errorf("APIKey = %q, want sk-ant-test", cfg.APIKey)
	}
	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("RequestTimeout = %v, want 30s", cfg.RequestTimeout)
	}
	if cfg.Temperature != 0.5 {
		t.Errorf("Temperature = %v, want 0.5", cfg.Temperature)
	}
	if cfg.MaxTokens != 8192 {
		t.Errorf("MaxTokens = %d, want 8192", cfg.MaxTokens)
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := Config{
		Provider:  "anthropic",
		Model:     "claude-sonnet-4-5-20250929",
		APIKey:    "sk-ant-test",
		MaxTokens: 4096,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestConfig_Validate_AllProviders(t *testing.T) {
	for _, provider := range []string{"anthropic", "openai", "gemini"} {
		t.Run(provider, func(t *testing.T) {
			cfg := Config{
				Provider:  provider,
				Model:     "test-model",
				APIKey:    "test-key",
				MaxTokens: 1024,
			}
			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil for provider %q", err, provider)
			}
		})
	}
}

func TestConfig_Validate_MissingProvider(t *testing.T) {
	cfg := Config{
		Provider:  "",
		Model:     "test-model",
		APIKey:    "test-key",
		MaxTokens: 1024,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for missing provider")
	}
}

func TestConfig_Validate_InvalidProvider(t *testing.T) {
	cfg := Config{
		Provider:  "llama",
		Model:     "test-model",
		APIKey:    "test-key",
		MaxTokens: 1024,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for invalid provider")
	}
}

func TestConfig_Validate_MissingModel(t *testing.T) {
	cfg := Config{
		Provider:  "anthropic",
		Model:     "",
		APIKey:    "test-key",
		MaxTokens: 1024,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for missing model")
	}
}

func TestConfig_Validate_MissingAPIKey(t *testing.T) {
	cfg := Config{
		Provider:  "anthropic",
		Model:     "test-model",
		APIKey:    "",
		MaxTokens: 1024,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for missing api_key")
	}
}

func TestConfig_Validate_ZeroMaxTokens(t *testing.T) {
	cfg := Config{
		Provider:  "anthropic",
		Model:     "test-model",
		APIKey:    "test-key",
		MaxTokens: 0,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for zero max_tokens")
	}
}
