# Functional Options Pattern

## What
A pattern for configuring structs using variadic function arguments instead of a config struct or constructor with many parameters. Each option is a function that modifies an options struct.

## Why
When a constructor has many optional parameters (level, color, timestamps, caller, output), functional options keep the API clean. Callers only specify what they care about. Defaults are sensible. New options can be added without breaking existing call sites.

## Example
```go
type Options struct {
    Level  Level
    Color  bool
    Output io.Writer
}

func DefaultOptions() Options {
    return Options{Level: LevelInfo, Color: true, Output: os.Stdout}
}

type Option func(*Options)

func WithLevel(l Level) Option {
    return func(o *Options) { o.Level = l }
}

// Constructor applies options over defaults.
func NewConsole(opts ...Option) Logger {
    o := DefaultOptions()
    for _, opt := range opts {
        opt(&o)
    }
    return &consoleLogger{opts: o}
}

// Usage — clean, only specify what you need.
log := NewConsole(WithLevel(LevelDebug), WithColor(false))
```

## Gotchas
- Define `DefaultOptions()` explicitly — don't rely on zero values being sensible.
- Each `With*` function closes over the value, so it's safe to build options dynamically.
- Don't confuse with the builder pattern — options configure at construction time, builders mutate after construction.

## Related
[[builder-pattern-errors]]
