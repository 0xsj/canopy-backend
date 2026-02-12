package auth

import (
	"github.com/0xsj/canopy-backend/pkg/config"
)

// Config holds authentication settings for token validation.
// Implements config.Section for use with the config loader.
//
// Environment variables (under CANOPY_AUTH_ prefix):
//
//	CANOPY_AUTH_ISSUER   — expected token issuer URL (e.g., "https://canopy.us.auth0.com/")
//	CANOPY_AUTH_AUDIENCE — expected audience claim (e.g., "https://api.canopy.dev")
//	CANOPY_AUTH_JWKS_URL — JWKS endpoint for public key retrieval
type Config struct {
	Issuer   string
	Audience string
	JWKSURL  string
}

func (c *Config) Load(env config.EnvReader) {
	c.Issuer = env.String("ISSUER", "")
	c.Audience = env.String("AUDIENCE", "")
	c.JWKSURL = env.String("JWKS_URL", "")
}

func (c *Config) Validate() error {
	var v config.Errors
	v.Required("issuer", c.Issuer)
	v.Required("audience", c.Audience)
	v.Required("jwks_url", c.JWKSURL)
	return v.Err()
}
