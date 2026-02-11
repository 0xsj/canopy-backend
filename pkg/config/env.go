package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvReader reads environment variables with a prefix.
// The prefix is prepended to all key lookups, producing
// keys like "CANOPY_SERVER_PORT" from prefix "CANOPY_SERVER" and key "PORT".
type EnvReader struct {
	prefix string
}

// NewEnvReader creates a reader with the given prefix.
// An empty prefix reads variables without any prefix.
func NewEnvReader(prefix string) EnvReader {
	return EnvReader{prefix: prefix}
}

// WithPrefix returns a new EnvReader with an additional prefix segment.
// NewEnvReader("CANOPY").WithPrefix("SERVER") reads "CANOPY_SERVER_*".
func (e EnvReader) WithPrefix(segment string) EnvReader {
	if e.prefix == "" {
		return EnvReader{prefix: segment}
	}
	return EnvReader{prefix: e.prefix + "_" + segment}
}

func (e EnvReader) key(name string) string {
	if e.prefix == "" {
		return name
	}
	return e.prefix + "_" + name
}

// String reads a string env var. Returns fallback if not set or empty.
func (e EnvReader) String(name, fallback string) string {
	val := os.Getenv(e.key(name))
	if val == "" {
		return fallback
	}
	return val
}

// Int reads an integer env var. Returns fallback if not set or unparseable.
func (e EnvReader) Int(name string, fallback int) int {
	val := os.Getenv(e.key(name))
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}

// Bool reads a boolean env var. Returns fallback if not set or unparseable.
// Truthy values: "true", "1", "yes". Case-insensitive.
func (e EnvReader) Bool(name string, fallback bool) bool {
	val := os.Getenv(e.key(name))
	if val == "" {
		return fallback
	}
	switch strings.ToLower(val) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return fallback
	}
}

// Duration reads a duration env var (e.g., "5s", "100ms").
// Returns fallback if not set or unparseable.
func (e EnvReader) Duration(name string, fallback time.Duration) time.Duration {
	val := os.Getenv(e.key(name))
	if val == "" {
		return fallback
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return d
}

// StringSlice reads a comma-separated env var into a string slice.
// Returns fallback if not set or empty.
func (e EnvReader) StringSlice(name string, fallback []string) []string {
	val := os.Getenv(e.key(name))
	if val == "" {
		return fallback
	}
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

// Require reads a string env var and records an error if not set.
// Use this for variables that must be present (secrets, DSNs, etc.).
func (e EnvReader) Require(name string) (string, error) {
	val := os.Getenv(e.key(name))
	if val == "" {
		return "", &MissingEnvError{Key: e.key(name)}
	}
	return val, nil
}

// MissingEnvError indicates a required environment variable was not set.
type MissingEnvError struct {
	Key string
}

func (e *MissingEnvError) Error() string {
	return "required environment variable not set: " + e.Key
}
