package httpserver

import (
	"testing"
	"time"

	"github.com/0xsj/canopy-backend/pkg/config"
)

func TestConfig_Load_Defaults(t *testing.T) {
	var cfg Config
	env := config.NewEnvReader("TEST_HTTP")
	cfg.Load(env)

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("ReadTimeout = %v, want 15s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 15*time.Second {
		t.Errorf("WriteTimeout = %v, want 15s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want 60s", cfg.IdleTimeout)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
}

func TestConfig_Load_FromEnv(t *testing.T) {
	t.Setenv("TEST_HTTP_HOST", "127.0.0.1")
	t.Setenv("TEST_HTTP_PORT", "9090")

	var cfg Config
	env := config.NewEnvReader("TEST_HTTP")
	cfg.Load(env)

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want 127.0.0.1", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := Config{Host: "0.0.0.0", Port: 8080}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestConfig_Validate_MissingHost(t *testing.T) {
	cfg := Config{Host: "", Port: 8080}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for missing host")
	}
}

func TestConfig_Validate_InvalidPort(t *testing.T) {
	cfg := Config{Host: "0.0.0.0", Port: 0}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for invalid port")
	}
}
