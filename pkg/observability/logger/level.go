package logger

import "strings"

// Level controls which messages are emitted.
// Messages below the configured level are discarded.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var levelStrings = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

func (l Level) String() string {
	if str, ok := levelStrings[l]; ok {
		return str
	}
	return "UNKNOWN"
}

// ParseLevel converts a string to a Level. Case-insensitive.
// Returns LevelInfo if the string is not recognized.
func ParseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}
