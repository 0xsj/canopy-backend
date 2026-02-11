package database

import (
	"testing"
	"time"

	"github.com/0xsj/canopy-backend/pkg/config"
)

func TestConfig_Load_Defaults(t *testing.T) {
	var cfg Config
	env := config.NewEnvReader("TEST_DB")
	cfg.Load(env)

	if cfg.DSN != "postgres://canopy:canopy@localhost:5433/canopy?sslmode=disable" {
		t.Errorf("DSN = %q, want default", cfg.DSN)
	}
	if cfg.MaxPoolSize != 20 {
		t.Errorf("MaxPoolSize = %d, want 20", cfg.MaxPoolSize)
	}
	if cfg.MinPoolSize != 5 {
		t.Errorf("MinPoolSize = %d, want 5", cfg.MinPoolSize)
	}
	if cfg.MaxConnLifetime != 30*time.Minute {
		t.Errorf("MaxConnLifetime = %v, want 30m", cfg.MaxConnLifetime)
	}
	if cfg.MaxConnIdleTime != 5*time.Minute {
		t.Errorf("MaxConnIdleTime = %v, want 5m", cfg.MaxConnIdleTime)
	}
}

func TestConfig_Load_FromEnv(t *testing.T) {
	t.Setenv("TEST_DB_DSN", "postgres://other:secret@remotehost:5432/otherdb")
	t.Setenv("TEST_DB_MAX_POOL_SIZE", "50")
	t.Setenv("TEST_DB_MIN_POOL_SIZE", "10")

	var cfg Config
	env := config.NewEnvReader("TEST_DB")
	cfg.Load(env)

	if cfg.DSN != "postgres://other:secret@remotehost:5432/otherdb" {
		t.Errorf("DSN = %q, want overridden value", cfg.DSN)
	}
	if cfg.MaxPoolSize != 50 {
		t.Errorf("MaxPoolSize = %d, want 50", cfg.MaxPoolSize)
	}
	if cfg.MinPoolSize != 10 {
		t.Errorf("MinPoolSize = %d, want 10", cfg.MinPoolSize)
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := Config{
		DSN:         "postgres://canopy:canopy@localhost:5433/canopy",
		MaxPoolSize: 20,
		MinPoolSize: 5,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestConfig_Validate_MissingDSN(t *testing.T) {
	cfg := Config{
		DSN:         "",
		MaxPoolSize: 20,
		MinPoolSize: 5,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for missing DSN")
	}
}

func TestConfig_Validate_ZeroPoolSize(t *testing.T) {
	cfg := Config{
		DSN:         "postgres://localhost/test",
		MaxPoolSize: 0,
		MinPoolSize: 5,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for zero MaxPoolSize")
	}
}

func TestConfig_Validate_NegativeMinPool(t *testing.T) {
	cfg := Config{
		DSN:         "postgres://localhost/test",
		MaxPoolSize: 20,
		MinPoolSize: -1,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for negative MinPoolSize")
	}
}

func TestConfig_SanitizedDSN(t *testing.T) {
	cfg := Config{
		DSN: "postgres://canopy:supersecret@localhost:5433/canopy?sslmode=disable",
	}

	got := cfg.sanitizedDSN()
	if got != "canopy@localhost:5433/canopy" {
		t.Errorf("sanitizedDSN() = %q, want password-free form", got)
	}
}

func TestConfig_SanitizedDSN_InvalidDSN(t *testing.T) {
	cfg := Config{
		DSN: "not-a-valid-dsn",
	}

	got := cfg.sanitizedDSN()
	if got != "<invalid>" {
		t.Errorf("sanitizedDSN() = %q, want %q", got, "<invalid>")
	}
}
