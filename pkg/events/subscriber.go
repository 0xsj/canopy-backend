package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Handler processes a received event.
// Return nil to acknowledge, return an error to trigger redelivery.
type Handler func(ctx context.Context, event Event) error

// Subscriber is the port for consuming domain events.
// Bounded context services accept this interface.
type Subscriber interface {
	Subscribe(ctx context.Context, subject string, handler Handler, opts ...SubscribeOption) (Subscription, error)
}

// Subscription represents an active event subscription.
type Subscription interface {
	Unsubscribe()
}

// SubscribeOption configures a subscription.
type SubscribeOption func(*subscribeConfig)

type subscribeConfig struct {
	consumerName string
	durable      bool
}

// WithConsumer sets a durable consumer name.
// Durable consumers survive restarts and resume from the last acknowledged message.
func WithConsumer(name string) SubscribeOption {
	return func(c *subscribeConfig) {
		c.consumerName = name
		c.durable = true
	}
}

type subscriber struct {
	stream jetstream.Stream
	log    logger.Logger
}

// NewSubscriber creates a Subscriber backed by NATS JetStream.
func NewSubscriber(broker *Broker) Subscriber {
	return &subscriber{
		stream: broker.stream,
		log:    broker.log,
	}
}

func (s *subscriber) Subscribe(ctx context.Context, subject string, handler Handler, opts ...SubscribeOption) (Subscription, error) {
	cfg := subscribeConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	consumerCfg := jetstream.ConsumerConfig{
		FilterSubjects: []string{subject},
		AckPolicy:      jetstream.AckExplicitPolicy,
	}

	if cfg.durable {
		consumerCfg.Durable = cfg.consumerName
	}

	consumer, err := s.stream.CreateOrUpdateConsumer(ctx, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("events: create consumer for %s: %w", subject, err)
	}

	log := s.log
	cons, err := consumer.Consume(func(msg jetstream.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			log.Error("event unmarshal failed",
				logger.String("subject", msg.Subject()),
				logger.Err(err),
			)
			msg.Term()
			return
		}

		if err := handler(context.Background(), event); err != nil {
			log.Warn("event handler failed",
				logger.String("type", event.Type),
				logger.String("id", event.ID),
				logger.Err(err),
			)
			msg.Nak()
			return
		}

		msg.Ack()
	})
	if err != nil {
		return nil, fmt.Errorf("events: consume %s: %w", subject, err)
	}

	s.log.Info("subscribed",
		logger.String("subject", subject),
		logger.String("consumer", cfg.consumerName),
	)

	return &subscription{cons: cons}, nil
}

type subscription struct {
	cons jetstream.ConsumeContext
}

func (s *subscription) Unsubscribe() {
	s.cons.Stop()
}
