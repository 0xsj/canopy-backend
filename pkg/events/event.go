package events

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Event is the envelope for all domain events published to NATS.
// Every bounded context publishes and consumes these.
type Event struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Subject     string          `json:"subject"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Data        json.RawMessage `json:"data"`
}

// New creates a new event with a generated ID and current timestamp.
// The subject is set by the publisher before sending.
func New(eventType string, workspaceID string, data any) (Event, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Event{}, fmt.Errorf("events: marshal data: %w", err)
	}

	return Event{
		ID:          generateID(),
		Type:        eventType,
		WorkspaceID: workspaceID,
		OccurredAt:  time.Now().UTC(),
		Data:        raw,
	}, nil
}

// Decode unmarshals the event Data into the given target.
func (e Event) Decode(target any) error {
	return json.Unmarshal(e.Data, target)
}

// generateID produces a 32-character hex string (16 random bytes).
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
