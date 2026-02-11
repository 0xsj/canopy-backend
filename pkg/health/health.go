package health

import (
	"context"
	"fmt"
	"sync"
)

// Checker is the port for any component that can report its health.
// Database, NATS broker, Redis — anything that can go down.
type Checker interface {
	Check(ctx context.Context) error
}

// CheckFunc adapts a plain function to the Checker interface.
//
// Usage:
//
//	monitor.Register("database", health.CheckFunc(db.Health))
//	monitor.Register("nats", health.CheckFunc(func(ctx context.Context) error {
//	    return broker.Health()
//	}))
type CheckFunc func(ctx context.Context) error

func (f CheckFunc) Check(ctx context.Context) error { return f(ctx) }

// Status is the result of a single component health check.
type Status struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Report is the aggregate result of all health checks.
type Report struct {
	Status     string   `json:"status"`
	Components []Status `json:"components"`
}

// IsHealthy returns true if all components are up.
func (r Report) IsHealthy() bool {
	return r.Status == "healthy"
}

// Monitor aggregates health checks from multiple components.
// Register components at startup, then call Check or Ready from HTTP handlers.
type Monitor struct {
	mu     sync.RWMutex
	checks []entry
}

type entry struct {
	name    string
	checker Checker
}

// NewMonitor creates an empty health monitor.
func NewMonitor() *Monitor {
	return &Monitor{}
}

// Register adds a named health checker.
// Call this during wiring, before the server starts.
func (m *Monitor) Register(name string, checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks = append(m.checks, entry{name: name, checker: checker})
}

// Liveness returns nil if the process is running.
// This is always true — if the code executes, the process is alive.
func (m *Monitor) Liveness() error {
	return nil
}

// Readiness checks all registered components concurrently.
// Returns a Report with per-component status.
func (m *Monitor) Readiness(ctx context.Context) Report {
	m.mu.RLock()
	checks := make([]entry, len(m.checks))
	copy(checks, m.checks)
	m.mu.RUnlock()

	statuses := make([]Status, len(checks))
	var wg sync.WaitGroup

	for i, c := range checks {
		wg.Add(1)
		go func(idx int, e entry) {
			defer wg.Done()
			s := Status{Name: e.name, Status: "up"}
			if err := e.checker.Check(ctx); err != nil {
				s.Status = "down"
				s.Error = err.Error()
			}
			statuses[idx] = s
		}(i, c)
	}

	wg.Wait()

	report := Report{Status: "healthy", Components: statuses}
	for _, s := range statuses {
		if s.Status == "down" {
			report.Status = "unhealthy"
			break
		}
	}

	return report
}

// ReadinessError returns nil if all components are healthy,
// or an error describing what's down.
func (m *Monitor) ReadinessError(ctx context.Context) error {
	report := m.Readiness(ctx)
	if report.IsHealthy() {
		return nil
	}

	var down []string
	for _, c := range report.Components {
		if c.Status == "down" {
			down = append(down, fmt.Sprintf("%s: %s", c.Name, c.Error))
		}
	}
	return fmt.Errorf("unhealthy: %v", down)
}
