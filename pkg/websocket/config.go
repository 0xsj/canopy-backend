package websocket

import (
	"time"

	"github.com/0xsj/canopy-backend/pkg/config"
)

// Config holds WebSocket server settings.
// Implements config.Section for use with the config loader.
type Config struct {
	PingInterval   time.Duration
	WriteTimeout   time.Duration
	ReadLimit      int
	SendBufferSize int
}

func (c *Config) Load(env config.EnvReader) {
	c.PingInterval = env.Duration("PING_INTERVAL", 30*time.Second)
	c.WriteTimeout = env.Duration("WRITE_TIMEOUT", 10*time.Second)
	c.ReadLimit = env.Int("READ_LIMIT", 32768) // 32KB
	c.SendBufferSize = env.Int("SEND_BUFFER_SIZE", 256)
}

func (c *Config) Validate() error {
	var v config.Errors
	v.Positive("send_buffer_size", c.SendBufferSize)
	return v.Err()
}
