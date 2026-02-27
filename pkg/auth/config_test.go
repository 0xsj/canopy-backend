package auth

import (
	"os"
	"testing"

	"github.com/0xsj/canopy-backend/pkg/config"
)

func TestConfig_Load(t *testing.T) {
	os.Setenv("TEST_AUTH_MODE", "jwks")
	os.Setenv("TEST_AUTH_ISSUER", "https://canopy.us.auth0.com/")
	os.Setenv("TEST_AUTH_AUDIENCE", "https://api.canopy.dev")
	os.Setenv("TEST_AUTH_JWKS_URL", "https://canopy.us.auth0.com/.well-known/jwks.json")
	defer func() {
		os.Unsetenv("TEST_AUTH_MODE")
		os.Unsetenv("TEST_AUTH_ISSUER")
		os.Unsetenv("TEST_AUTH_AUDIENCE")
		os.Unsetenv("TEST_AUTH_JWKS_URL")
	}()

	var cfg Config
	env := config.NewEnvReader("TEST_AUTH")
	cfg.Load(env)

	if cfg.Mode != "jwks" {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "jwks")
	}
	if cfg.Issuer != "https://canopy.us.auth0.com/" {
		t.Errorf("Issuer = %q, want %q", cfg.Issuer, "https://canopy.us.auth0.com/")
	}
	if cfg.Audience != "https://api.canopy.dev" {
		t.Errorf("Audience = %q, want %q", cfg.Audience, "https://api.canopy.dev")
	}
	if cfg.JWKSURL != "https://canopy.us.auth0.com/.well-known/jwks.json" {
		t.Errorf("JWKSURL = %q, want %q", cfg.JWKSURL, "https://canopy.us.auth0.com/.well-known/jwks.json")
	}
}

func TestConfig_Validate_JWKS(t *testing.T) {
	cfg := Config{
		Mode:    "jwks",
		Issuer:  "https://canopy.us.auth0.com/",
		JWKSURL: "https://canopy.us.auth0.com/.well-known/jwks.json",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestConfig_Validate_Dev(t *testing.T) {
	cfg := Config{Mode: "dev"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for dev mode", err)
	}
}

func TestConfig_Validate_DefaultIsDev(t *testing.T) {
	var cfg Config
	env := config.NewEnvReader("NONEXISTENT_PREFIX")
	cfg.Load(env)
	if cfg.Mode != "dev" {
		t.Errorf("Mode = %q, want %q (default)", cfg.Mode, "dev")
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for default dev mode", err)
	}
}

func TestConfig_Validate_JWKS_MissingIssuer(t *testing.T) {
	cfg := Config{
		Mode:    "jwks",
		JWKSURL: "https://example.com/.well-known/jwks.json",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error for missing issuer")
	}
}

func TestConfig_Validate_JWKS_MissingJWKSURL(t *testing.T) {
	cfg := Config{
		Mode:   "jwks",
		Issuer: "https://canopy.us.auth0.com/",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error for missing jwks_url")
	}
}

func TestConfig_IsDev(t *testing.T) {
	dev := Config{Mode: "dev"}
	if !dev.IsDev() {
		t.Error("IsDev() = false, want true")
	}
	jwks := Config{Mode: "jwks"}
	if jwks.IsDev() {
		t.Error("IsDev() = true, want false")
	}
}
