package events

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Broker manages the NATS connection and JetStream context.
// It is the single point of NATS wiring — passed via constructor injection.
type Broker struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
	log    logger.Logger
	cfg    Config
}

// Connect establishes a NATS connection and initializes JetStream.
// Creates or updates the event stream on first connect.
func Connect(ctx context.Context, cfg Config, log logger.Logger) (*Broker, error) {
	opts := []nats.Option{
		nats.Name("canopy"),
		nats.Timeout(cfg.ConnectTimeout),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Warn("nats disconnected", logger.Err(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info("nats reconnected", logger.String("url", nc.ConnectedUrl()))
		}),
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("events: connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("events: jetstream init: %w", err)
	}

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      cfg.StreamName,
		Subjects:  []string{"workspace.>"},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("events: create stream: %w", err)
	}

	log.Info("nats connected",
		logger.String("url", nc.ConnectedUrl()),
		logger.String("stream", cfg.StreamName),
	)

	return &Broker{
		conn:   nc,
		js:     js,
		stream: stream,
		log:    log,
		cfg:    cfg,
	}, nil
}

// JetStream returns the underlying JetStream context for advanced usage.
func (b *Broker) JetStream() jetstream.JetStream {
	return b.js
}

// Stream returns the managed stream.
func (b *Broker) Stream() jetstream.Stream {
	return b.stream
}

// Health checks the NATS connection status.
func (b *Broker) Health() error {
	if b.conn.Status() != nats.CONNECTED {
		return fmt.Errorf("events: nats status %d (not connected)", b.conn.Status())
	}
	return nil
}

// Close drains the connection (finishes in-flight messages) and closes.
func (b *Broker) Close() error {
	if err := b.conn.Drain(); err != nil {
		b.log.Warn("nats drain failed, forcing close", logger.Err(err))
		b.conn.Close()
		return nil
	}
	b.log.Info("nats connection closed")
	return nil
}
