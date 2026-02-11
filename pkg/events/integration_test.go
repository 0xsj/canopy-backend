//go:build integration

package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

const testNATSURL = "nats://localhost:4223"

func testBroker(t *testing.T) *Broker {
	t.Helper()
	ctx := context.Background()
	cfg := Config{
		URL:            testNATSURL,
		StreamName:     "canopy_test",
		ConnectTimeout: 5 * time.Second,
		ReconnectWait:  1 * time.Second,
		MaxReconnects:  3,
	}
	broker, err := Connect(ctx, cfg, logger.NewNoop())
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	t.Cleanup(func() { broker.Close() })
	return broker
}

// --- Connect ---

func TestConnect_Success(t *testing.T) {
	broker := testBroker(t)
	if broker.JetStream() == nil {
		t.Error("JetStream() is nil after Connect")
	}
	if broker.Stream() == nil {
		t.Error("Stream() is nil after Connect")
	}
}

func TestConnect_InvalidURL(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		URL:            "nats://nobody:59999",
		StreamName:     "test",
		ConnectTimeout: 1 * time.Second,
		MaxReconnects:  0,
	}
	_, err := Connect(ctx, cfg, logger.NewNoop())
	if err == nil {
		t.Fatal("Connect() = nil, want error for bad URL")
	}
}

// --- Health ---

func TestHealth_Success(t *testing.T) {
	broker := testBroker(t)
	if err := broker.Health(); err != nil {
		t.Errorf("Health() = %v, want nil", err)
	}
}

// --- Publish + Subscribe ---

func TestPublishSubscribe_Roundtrip(t *testing.T) {
	broker := testBroker(t)
	ctx := context.Background()

	pub := NewPublisher(broker)
	sub := NewSubscriber(broker)

	type leafData struct {
		Title string `json:"title"`
	}

	subject := BuildSubject("ws_test", "exploration", "leaf_created")

	var received Event
	var wg sync.WaitGroup
	wg.Add(1)

	subscription, err := sub.Subscribe(ctx, subject, func(_ context.Context, event Event) error {
		received = event
		wg.Done()
		return nil
	}, WithConsumer("test_roundtrip"))
	if err != nil {
		t.Fatalf("Subscribe() error: %v", err)
	}
	defer subscription.Unsubscribe()

	// Small delay for consumer to be ready.
	time.Sleep(100 * time.Millisecond)

	event, err := New("leaf_created", "ws_test", leafData{Title: "integration test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	event.Subject = subject

	if err := pub.Publish(ctx, event); err != nil {
		t.Fatalf("Publish() error: %v", err)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	if received.ID != event.ID {
		t.Errorf("received ID = %q, want %q", received.ID, event.ID)
	}
	if received.Type != "leaf_created" {
		t.Errorf("received Type = %q, want leaf_created", received.Type)
	}

	var data leafData
	if err := received.Decode(&data); err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if data.Title != "integration test" {
		t.Errorf("data.Title = %q, want %q", data.Title, "integration test")
	}
}

func TestPublish_MissingSubject(t *testing.T) {
	broker := testBroker(t)
	pub := NewPublisher(broker)

	event, _ := New("test", "ws_1", nil)
	// event.Subject is empty

	err := pub.Publish(context.Background(), event)
	if err == nil {
		t.Error("Publish() = nil, want error for missing subject")
	}
}

func TestSubscribe_HandlerError_Naks(t *testing.T) {
	broker := testBroker(t)
	ctx := context.Background()

	pub := NewPublisher(broker)
	sub := NewSubscriber(broker)

	subject := BuildSubject("ws_test", "exploration", "handler_error")

	attempts := 0
	var mu sync.Mutex
	var once sync.Once
	done := make(chan struct{})

	subscription, err := sub.Subscribe(ctx, subject, func(_ context.Context, event Event) error {
		mu.Lock()
		attempts++
		count := attempts
		mu.Unlock()

		if count >= 2 {
			once.Do(func() { close(done) })
			return nil
		}
		return errors.New("intentional failure")
	}, WithConsumer("test_nak"))
	if err != nil {
		t.Fatalf("Subscribe() error: %v", err)
	}
	defer subscription.Unsubscribe()

	time.Sleep(100 * time.Millisecond)

	event, _ := New("handler_error", "ws_test", nil)
	event.Subject = subject
	pub.Publish(ctx, event)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for redelivery")
	}

	mu.Lock()
	if attempts < 2 {
		t.Errorf("attempts = %d, want >= 2 (message should be redelivered)", attempts)
	}
	mu.Unlock()
}
