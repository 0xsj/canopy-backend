package websocket

import (
	"encoding/json"
	"fmt"
)

// Message is the envelope for all WebSocket messages.
// Both client→server and server→client messages use this format.
type Message struct {
	Type        string          `json:"type"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
}

// NewMessage creates a message with the given type and payload.
func NewMessage(msgType string, workspaceID string, data any) (Message, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Message{}, fmt.Errorf("websocket: marshal message: %w", err)
	}
	return Message{
		Type:        msgType,
		WorkspaceID: workspaceID,
		Data:        raw,
	}, nil
}

// Decode unmarshals the message Data into the given target.
func (m Message) Decode(target any) error {
	return json.Unmarshal(m.Data, target)
}

// Encode serializes the message to JSON bytes.
func (m Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// DecodeMessage parses raw bytes into a Message.
func DecodeMessage(data []byte) (Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, fmt.Errorf("websocket: decode message: %w", err)
	}
	return msg, nil
}
