package websocket

import (
	"testing"
)

func TestNewMessage(t *testing.T) {
	msg, err := NewMessage("leaf_created", "ws_abc", map[string]string{"title": "hello"})
	if err != nil {
		t.Fatalf("NewMessage() error: %v", err)
	}
	if msg.Type != "leaf_created" {
		t.Errorf("Type = %q, want leaf_created", msg.Type)
	}
	if msg.WorkspaceID != "ws_abc" {
		t.Errorf("WorkspaceID = %q, want ws_abc", msg.WorkspaceID)
	}
	if msg.Data == nil {
		t.Error("Data is nil")
	}
}

func TestNewMessage_MarshalError(t *testing.T) {
	_, err := NewMessage("test", "ws_1", make(chan int))
	if err == nil {
		t.Error("NewMessage() = nil, want marshal error")
	}
}

func TestMessage_Decode(t *testing.T) {
	type payload struct {
		Title string `json:"title"`
	}

	msg, _ := NewMessage("test", "ws_1", payload{Title: "brainstorm"})

	var decoded payload
	if err := msg.Decode(&decoded); err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if decoded.Title != "brainstorm" {
		t.Errorf("Title = %q, want brainstorm", decoded.Title)
	}
}

func TestMessage_Encode_Decode_Roundtrip(t *testing.T) {
	original, _ := NewMessage("leaf_created", "ws_roundtrip", map[string]int{"points": 5})

	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	decoded, err := DecodeMessage(data)
	if err != nil {
		t.Fatalf("DecodeMessage() error: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, original.Type)
	}
	if decoded.WorkspaceID != original.WorkspaceID {
		t.Errorf("WorkspaceID = %q, want %q", decoded.WorkspaceID, original.WorkspaceID)
	}
}

func TestDecodeMessage_Invalid(t *testing.T) {
	_, err := DecodeMessage([]byte("not json"))
	if err == nil {
		t.Error("DecodeMessage() = nil, want error")
	}
}
