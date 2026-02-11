package logger

import (
	"io"
	"os"
)

// Options configures a console logger.
type Options struct {
	Level      Level
	Color      bool
	Timestamps bool
	Caller     bool
	Output     io.Writer
}

// DefaultOptions returns sensible dev-mode defaults:
// INFO level, colorized, timestamps on, caller off, stdout.
func DefaultOptions() Options {
	return Options{
		Level:      LevelInfo,
		Color:      true,
		Timestamps: true,
		Caller:     false,
		Output:     os.Stdout,
	}
}

// Option is a functional option for configuring the console logger.
type Option func(*Options)

func WithLevel(l Level) Option {
	return func(o *Options) { o.Level = l }
}

func WithColor(enabled bool) Option {
	return func(o *Options) { o.Color = enabled }
}

func WithTimestamps(enabled bool) Option {
	return func(o *Options) { o.Timestamps = enabled }
}

func WithCaller(enabled bool) Option {
	return func(o *Options) { o.Caller = enabled }
}

func WithOutput(w io.Writer) Option {
	return func(o *Options) { o.Output = w }
}
