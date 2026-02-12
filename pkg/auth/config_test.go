package auth

import (
	"os"
	"testing"

	"github.com/0xsj/canopy-backend/pkg/config"
)

func TestConfig_Load(t *testing.T) {
	os.Setenv("TEST_AUTH_ISSUER", "https://canopy.us.auth0.com/")
	os.Setenv("TEST_AUTH_AUDIENCE", "https://api.canopy.dev")
	os.Setenv("TEST_AUTH_JWKS_URL", "https://canopy.us.auth0.com/.well-known/jwks.json")
	defer func() {
		os.Unsetenv("TEST_AUTH_ISSUER")
		os.Unsetenv("TEST_AUTH_AUDIENCE")
		os.Unsetenv("TEST_AUTH_JWKS_URL")
	}()

	var cfg Config
	env := config.NewEnvReader("TEST_AUTH")
	cfg.Load(env)

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

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := Config{
		Issuer:   "https://canopy.us.auth0.com/",
		Audience: "https://api.canopy.dev",
		JWKSURL:  "https://canopy.us.auth0.com/.well-known/jwks.json",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestConfig_Validate_MissingAll(t *testing.T) {
	var cfg Config
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}
}

func TestConfig_Validate_MissingIssuer(t *testing.T) {
	cfg := Config{
		Audience: "https://api.canopy.dev",
		JWKSURL:  "https://example.com/.well-known/jwks.json",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want error for missing issuer")
	}
}
