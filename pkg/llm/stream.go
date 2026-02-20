package llm

// StreamBroadcaster is a port for delivering LLM streaming events to clients.
// Implementations send messages over WebSocket (or any real-time transport).
// Services depend on this interface, not on pkg/websocket directly.
type StreamBroadcaster interface {
	Send(workspaceID string, msg StreamMessage) error
}

// StreamMessage is the payload sent to clients during LLM streaming.
type StreamMessage struct {
	Type     string `json:"type"`              // stream.start | stream.chunk | stream.end | stream.error
	StreamID string `json:"stream_id"`         // session_id or synthesis_id
	Delta    string `json:"delta,omitempty"`   // incremental text (stream.chunk)
	Content  string `json:"content,omitempty"` // full text (stream.end)
	Error    string `json:"error,omitempty"`   // error description (stream.error)
}
