package llm

import (
	"context"
	"testing"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

func TestNewProvider_Anthropic(t *testing.T) {
	cfg := Config{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5-20250929",
		APIKey:   "sk-ant-test",
	}
	log := logger.NewNoop()

	p, err := NewProvider(context.Background(), cfg, log)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewProvider() returned nil")
	}
	if _, ok := p.(*anthropicProvider); !ok {
		t.Errorf("NewProvider() type = %T, want *anthropicProvider", p)
	}
}

func TestNewProvider_OpenAI(t *testing.T) {
	cfg := Config{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "sk-test",
	}
	log := logger.NewNoop()

	p, err := NewProvider(context.Background(), cfg, log)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewProvider() returned nil")
	}
	if _, ok := p.(*openaiProvider); !ok {
		t.Errorf("NewProvider() type = %T, want *openaiProvider", p)
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	cfg := Config{
		Provider: "llama",
		Model:    "test",
		APIKey:   "test",
	}
	log := logger.NewNoop()

	p, err := NewProvider(context.Background(), cfg, log)
	if err == nil {
		t.Error("NewProvider() error = nil, want error for unknown provider")
	}
	if p != nil {
		t.Errorf("NewProvider() = %v, want nil on error", p)
	}
}

func TestNewProvider_EmptyProvider(t *testing.T) {
	cfg := Config{
		Provider: "",
		Model:    "test",
		APIKey:   "test",
	}
	log := logger.NewNoop()

	p, err := NewProvider(context.Background(), cfg, log)
	if err == nil {
		t.Error("NewProvider() error = nil, want error for empty provider")
	}
	if p != nil {
		t.Errorf("NewProvider() = %v, want nil on error", p)
	}
}

func TestNewProvider_AdaptersImplementInterface(t *testing.T) {
	// Compile-time check that all adapters implement Provider.
	var _ Provider = (*anthropicProvider)(nil)
	var _ Provider = (*openaiProvider)(nil)
	var _ Provider = (*geminiProvider)(nil)
}
