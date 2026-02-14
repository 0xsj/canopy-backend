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
