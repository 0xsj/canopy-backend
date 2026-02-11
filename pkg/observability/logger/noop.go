package logger

// noopLogger discards all log output. Used in tests
// and as the fallback when no logger is in context.
type noopLogger struct{}

func NewNoop() Logger {
	return &noopLogger{}
}

func (n *noopLogger) Debug(string, ...Field) {}
func (n *noopLogger) Info(string, ...Field)  {}
func (n *noopLogger) Warn(string, ...Field)  {}
func (n *noopLogger) Error(string, ...Field) {}
func (n *noopLogger) With(...Field) Logger   { return n }
