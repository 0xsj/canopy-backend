package llm

import "context"

// Provider is the port for LLM interactions. Adapters implement this
// interface for each vendor (Anthropic, OpenAI, Gemini). The session
// context calls ChatCompletion after assembling prompt context.
type Provider interface {
	ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// ChatRequest carries the full prompt payload to the LLM provider.
type ChatRequest struct {
	Model    string
	Messages []Message
	Options  Options
}

// ChatResponse carries the LLM's reply and usage metadata.
type ChatResponse struct {
	Content string
	Model   string
	Usage   Usage
}

// Message represents a single turn in a conversation.
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// Options holds optional generation parameters.
type Options struct {
	Temperature float64
	MaxTokens   int
	TopP        float64
	Stop        []string
}

// Usage reports token consumption for metering and observability.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// StreamChunk carries a single token or completion signal from a streaming LLM call.
type StreamChunk struct {
	Delta string
	Done  bool
	Usage *Usage // non-nil only when Done
}

// StreamHandler is called for each chunk during a streaming LLM call.
type StreamHandler func(chunk StreamChunk) error

// StreamProvider is an optional capability for providers that support streaming.
// Services type-assert Provider to StreamProvider; non-streaming providers
// fall back to synchronous ChatCompletion automatically.
type StreamProvider interface {
	Provider
	ChatCompletionStream(ctx context.Context, req ChatRequest, handler StreamHandler) error
}
