package logger

import "fmt"

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

func colorize(color, text string) string {
	return fmt.Sprintf("%s%s%s", color, text, colorReset)
}

// levelColor returns the ANSI-colored level string.
func levelColor(level Level) string {
	switch level {
	case LevelDebug:
		return colorize(colorGray, "DEBUG")
	case LevelInfo:
		return colorize(colorGreen, " INFO")
	case LevelWarn:
		return colorize(colorYellow, " WARN")
	case LevelError:
		return colorize(colorRed, "ERROR")
	default:
		return level.String()
	}
}

// levelPlain returns the plain (no color) level string, padded to 5 chars.
func levelPlain(level Level) string {
	switch level {
	case LevelInfo:
		return " INFO"
	case LevelWarn:
		return " WARN"
	default:
		return level.String()
	}
}
