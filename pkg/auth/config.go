package auth

import (
	"github.com/0xsj/canopy-backend/pkg/config"
)

// Config holds authentication settings for token validation.
// Implements config.Section for use with the config loader.
//
// Environment variables (under CANOPY_AUTH_ prefix):
//
//	CANOPY_AUTH_MODE     — "dev" for static validator, "jwks" for production (default: "dev")
//	CANOPY_AUTH_ISSUER   — expected token issuer URL (e.g., "https://verb-noun-00.clerk.accounts.dev")
//	CANOPY_AUTH_AUDIENCE — expected audience claim (optional — Clerk omits aud by default)
//	CANOPY_AUTH_JWKS_URL — JWKS endpoint for public key retrieval
type Config struct {
	Mode     string // "dev" or "jwks"
	Issuer   string
	Audience string
	JWKSURL  string
}

func (c *Config) Load(env config.EnvReader) {
	c.Mode = env.String("MODE", "dev")
	c.Issuer = env.String("ISSUER", "")
	c.Audience = env.String("AUDIENCE", "")
	c.JWKSURL = env.String("JWKS_URL", "")
}

func (c *Config) Validate() error {
	var v config.Errors
	v.OneOf("mode", c.Mode, []string{"dev", "jwks"})
	if c.Mode == "jwks" {
		v.Required("issuer", c.Issuer)
		v.Required("jwks_url", c.JWKSURL)
		// audience is intentionally optional — Clerk does not set aud by default.
	}
	return v.Err()
}

// IsDev reports whether the config is set for local development.
func (c *Config) IsDev() bool {
	return c.Mode == "dev"
}
