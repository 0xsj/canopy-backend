package events

import (
	"testing"
	"time"

	"github.com/0xsj/canopy-backend/pkg/config"
)

func TestConfig_Load_Defaults(t *testing.T) {
	var cfg Config
	env := config.NewEnvReader("TEST_EVENTS")
	cfg.Load(env)

	if cfg.URL != "nats://localhost:4223" {
		t.Errorf("URL = %q, want default", cfg.URL)
	}
	if cfg.StreamName != "canopy" {
		t.Errorf("StreamName = %q, want canopy", cfg.StreamName)
	}
	if cfg.ConnectTimeout != 5*time.Second {
		t.Errorf("ConnectTimeout = %v, want 5s", cfg.ConnectTimeout)
	}
	if cfg.ReconnectWait != 2*time.Second {
		t.Errorf("ReconnectWait = %v, want 2s", cfg.ReconnectWait)
	}
	if cfg.MaxReconnects != 60 {
		t.Errorf("MaxReconnects = %d, want 60", cfg.MaxReconnects)
	}
}

func TestConfig_Load_FromEnv(t *testing.T) {
	t.Setenv("TEST_EVENTS_URL", "nats://remote:4222")
	t.Setenv("TEST_EVENTS_STREAM_NAME", "mystream")
	t.Setenv("TEST_EVENTS_MAX_RECONNECTS", "100")

	var cfg Config
	env := config.NewEnvReader("TEST_EVENTS")
	cfg.Load(env)

	if cfg.URL != "nats://remote:4222" {
		t.Errorf("URL = %q, want overridden value", cfg.URL)
	}
	if cfg.StreamName != "mystream" {
		t.Errorf("StreamName = %q, want mystream", cfg.StreamName)
	}
	if cfg.MaxReconnects != 100 {
		t.Errorf("MaxReconnects = %d, want 100", cfg.MaxReconnects)
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := Config{
		URL:        "nats://localhost:4222",
		StreamName: "canopy",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestConfig_Validate_MissingURL(t *testing.T) {
	cfg := Config{
		URL:        "",
		StreamName: "canopy",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for missing URL")
	}
}

func TestConfig_Validate_MissingStreamName(t *testing.T) {
	cfg := Config{
		URL:        "nats://localhost:4222",
		StreamName: "",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for missing StreamName")
	}
}
