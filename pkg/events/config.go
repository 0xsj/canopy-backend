package events

import (
	"time"

	"github.com/0xsj/canopy-backend/pkg/config"
)

// Config holds NATS connection and JetStream settings.
// Implements config.Section for use with the config loader.
type Config struct {
	URL            string
	StreamName     string
	ConnectTimeout time.Duration
	ReconnectWait  time.Duration
	MaxReconnects  int
}

func (c *Config) Load(env config.EnvReader) {
	c.URL = env.String("URL", "nats://localhost:4223")
	c.StreamName = env.String("STREAM_NAME", "canopy")
	c.ConnectTimeout = env.Duration("CONNECT_TIMEOUT", 5*time.Second)
	c.ReconnectWait = env.Duration("RECONNECT_WAIT", 2*time.Second)
	c.MaxReconnects = env.Int("MAX_RECONNECTS", 60)
}

func (c *Config) Validate() error {
	var v config.Errors
	v.Required("url", c.URL)
	v.Required("stream_name", c.StreamName)
	return v.Err()
}
