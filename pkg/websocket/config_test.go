package websocket

import (
	"testing"
	"time"

	"github.com/0xsj/canopy-backend/pkg/config"
)

func TestConfig_Load_Defaults(t *testing.T) {
	var cfg Config
	env := config.NewEnvReader("TEST_WS")
	cfg.Load(env)

	if cfg.PingInterval != 30*time.Second {
		t.Errorf("PingInterval = %v, want 30s", cfg.PingInterval)
	}
	if cfg.WriteTimeout != 10*time.Second {
		t.Errorf("WriteTimeout = %v, want 10s", cfg.WriteTimeout)
	}
	if cfg.ReadLimit != 32768 {
		t.Errorf("ReadLimit = %d, want 32768", cfg.ReadLimit)
	}
	if cfg.SendBufferSize != 256 {
		t.Errorf("SendBufferSize = %d, want 256", cfg.SendBufferSize)
	}
}

func TestConfig_Load_FromEnv(t *testing.T) {
	t.Setenv("TEST_WS_SEND_BUFFER_SIZE", "512")
	t.Setenv("TEST_WS_READ_LIMIT", "65536")

	var cfg Config
	env := config.NewEnvReader("TEST_WS")
	cfg.Load(env)

	if cfg.SendBufferSize != 512 {
		t.Errorf("SendBufferSize = %d, want 512", cfg.SendBufferSize)
	}
	if cfg.ReadLimit != 65536 {
		t.Errorf("ReadLimit = %d, want 65536", cfg.ReadLimit)
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := Config{SendBufferSize: 256}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestConfig_Validate_ZeroBuffer(t *testing.T) {
	cfg := Config{SendBufferSize: 0}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for zero SendBufferSize")
	}
}
