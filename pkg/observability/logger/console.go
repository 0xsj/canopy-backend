package logger

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// consoleLogger writes human-readable, optionally colorized log lines.
// It is the default logger for development.
type consoleLogger struct {
	opts   Options
	fields []Field
	mu     sync.Mutex
}

// NewConsole creates a console logger with the given options.
func NewConsole(opts ...Option) Logger {
	o := DefaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &consoleLogger{opts: o}
}

func (c *consoleLogger) Debug(msg string, fields ...Field) { c.log(LevelDebug, msg, fields) }
func (c *consoleLogger) Info(msg string, fields ...Field)  { c.log(LevelInfo, msg, fields) }
func (c *consoleLogger) Warn(msg string, fields ...Field)  { c.log(LevelWarn, msg, fields) }
func (c *consoleLogger) Error(msg string, fields ...Field) { c.log(LevelError, msg, fields) }

func (c *consoleLogger) With(fields ...Field) Logger {
	combined := make([]Field, 0, len(c.fields)+len(fields))
	combined = append(combined, c.fields...)
	combined = append(combined, fields...)
	return &consoleLogger{
		opts:   c.opts,
		fields: combined,
	}
}

func (c *consoleLogger) log(level Level, msg string, fields []Field) {
	if level < c.opts.Level {
		return
	}

	var b strings.Builder

	// Timestamp.
	if c.opts.Timestamps {
		ts := time.Now().Format("15:04:05.000")
		if c.opts.Color {
			b.WriteString(colorize(colorGray, ts))
		} else {
			b.WriteString(ts)
		}
		b.WriteByte(' ')
	}

	// Level.
	if c.opts.Color {
		b.WriteString(levelColor(level))
	} else {
		b.WriteString(levelPlain(level))
	}
	b.WriteByte(' ')

	// Caller.
	if c.opts.Caller {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			// Shorten to just filename:line.
			if idx := strings.LastIndex(file, "/"); idx >= 0 {
				file = file[idx+1:]
			}
			caller := fmt.Sprintf("%s:%d", file, line)
			if c.opts.Color {
				b.WriteString(colorize(colorGray, caller))
			} else {
				b.WriteString(caller)
			}
			b.WriteByte(' ')
		}
	}

	// Message.
	b.WriteString(msg)

	// Fields — preset fields first, then call-site fields.
	allFields := make([]Field, 0, len(c.fields)+len(fields))
	allFields = append(allFields, c.fields...)
	allFields = append(allFields, fields...)

	for _, f := range allFields {
		b.WriteByte(' ')
		if c.opts.Color {
			b.WriteString(colorize(colorCyan, f.Key))
			b.WriteByte('=')
			b.WriteString(f.FormatValue())
		} else {
			b.WriteString(f.Key)
			b.WriteByte('=')
			b.WriteString(f.FormatValue())
		}
	}

	b.WriteByte('\n')

	c.mu.Lock()
	fmt.Fprint(c.opts.Output, b.String())
	c.mu.Unlock()
}
