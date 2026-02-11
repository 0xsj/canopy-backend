package logger

import "context"

// Logger is the port for structured logging throughout Canopy.
// All components accept a Logger via constructor injection.
// Implementations are swappable: noop for tests, console for dev, zap/slog for production.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)

	// With returns a child logger with the given fields preset on every entry.
	With(fields ...Field) Logger
}

type contextKey struct{}

// WithContext stores a Logger in the context.
// Middleware uses this to attach a logger with request-scoped fields (e.g., request ID).
func WithContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext extracts a Logger from the context.
// Returns a Noop logger if none is found.
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(contextKey{}).(Logger); ok {
		return l
	}
	return NewNoop()
}
