package config

import (
	"fmt"
	"strings"
)

// Section is the interface every config section implements.
// Each bounded context or infrastructure concern defines its own Section.
type Section interface {
	// Load reads values from the environment using the provided reader.
	// The reader is already scoped to this section's prefix.
	Load(env EnvReader)

	// Validate checks that the loaded config is valid.
	// Returns nil if everything is fine, an error describing what's wrong otherwise.
	Validate() error
}

// entry pairs a section with its prefix name.
type entry struct {
	name    string
	section Section
}

// Loader composes config sections, loads them from environment variables,
// and validates them all at startup. This is the single point of config wiring.
type Loader struct {
	prefix  string
	entries []entry
}

// NewLoader creates a config loader with the given root prefix.
// All env vars will be prefixed with this value (e.g., "CANOPY").
func NewLoader(prefix string) *Loader {
	return &Loader{prefix: prefix}
}

// Register adds a config section under a named prefix.
// The section will read from "{root}_{name}_*" env vars.
// For example, Register("SERVER", &serverCfg) reads "CANOPY_SERVER_PORT".
func (l *Loader) Register(name string, section Section) {
	l.entries = append(l.entries, entry{
		name:    strings.ToUpper(name),
		section: section,
	})
}

// LoadAll loads and validates all registered sections.
// Returns a combined error if any section fails validation.
// This is called once at startup — fail fast on any misconfiguration.
func (l *Loader) LoadAll() error {
	var errs []string

	for _, e := range l.entries {
		reader := NewEnvReader(l.prefix).WithPrefix(e.name)
		e.section.Load(reader)

		if err := e.section.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", strings.ToLower(e.name), err.Error()))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}
