package crypto

import (
	"encoding/hex"
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/config"
)

// Config holds encryption settings.
// Registered as the "CRYPTO" section: reads CANOPY_CRYPTO_KEY.
type Config struct {
	Key string // hex-encoded 32-byte key (64 hex chars)
}

func (c *Config) Load(env config.EnvReader) {
	c.Key = env.String("KEY", "")
}

func (c *Config) Validate() error {
	var v config.Errors
	v.Required("key", c.Key)
	if c.Key != "" {
		b, err := hex.DecodeString(c.Key)
		if err != nil {
			v.Add("key must be valid hex: %v", err)
		} else if len(b) != 32 {
			v.Add("key must be exactly 32 bytes (64 hex chars), got %d bytes", len(b))
		}
	}
	return v.Err()
}

// KeyBytes decodes the hex key into a 32-byte array.
// Must only be called after successful Validate().
func (c *Config) KeyBytes() ([32]byte, error) {
	b, err := hex.DecodeString(c.Key)
	if err != nil {
		return [32]byte{}, fmt.Errorf("crypto: decode key: %w", err)
	}
	var key [32]byte
	copy(key[:], b)
	return key, nil
}
