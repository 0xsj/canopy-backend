package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Publisher is the port for publishing domain events.
// Bounded context services accept this interface.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

type publisher struct {
	js  jetstream.JetStream
	log logger.Logger
}

// NewPublisher creates a Publisher backed by NATS JetStream.
func NewPublisher(broker *Broker) Publisher {
	return &publisher{
		js:  broker.js,
		log: broker.log,
	}
}

func (p *publisher) Publish(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("events: marshal event: %w", err)
	}

	if event.Subject == "" {
		return fmt.Errorf("events: subject is required")
	}

	ack, err := p.js.Publish(ctx, event.Subject, data)
	if err != nil {
		return fmt.Errorf("events: publish %s: %w", event.Type, err)
	}

	p.log.Debug("event published",
		logger.String("type", event.Type),
		logger.String("subject", event.Subject),
		logger.String("id", event.ID),
		logger.Any("seq", ack.Sequence),
	)

	return nil
}
