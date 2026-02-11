package httpserver

import (
	"time"

	"github.com/0xsj/canopy-backend/pkg/config"
)

// Config holds HTTP server settings.
// Implements config.Section for use with the config loader.
type Config struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func (c *Config) Load(env config.EnvReader) {
	c.Host = env.String("HOST", "0.0.0.0")
	c.Port = env.Int("PORT", 8080)
	c.ReadTimeout = env.Duration("READ_TIMEOUT", 15*time.Second)
	c.WriteTimeout = env.Duration("WRITE_TIMEOUT", 15*time.Second)
	c.IdleTimeout = env.Duration("IDLE_TIMEOUT", 60*time.Second)
	c.ShutdownTimeout = env.Duration("SHUTDOWN_TIMEOUT", 10*time.Second)
}

func (c *Config) Validate() error {
	var v config.Errors
	v.Required("host", c.Host)
	v.PortRange("port", c.Port)
	return v.Err()
}
