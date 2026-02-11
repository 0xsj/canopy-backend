package health

import (
	"context"
	"errors"
	"testing"
)

func TestMonitor_Readiness_AllHealthy(t *testing.T) {
	m := NewMonitor()
	m.Register("db", CheckFunc(func(ctx context.Context) error { return nil }))
	m.Register("nats", CheckFunc(func(ctx context.Context) error { return nil }))

	report := m.Readiness(context.Background())
	if !report.IsHealthy() {
		t.Errorf("Status = %q, want healthy", report.Status)
	}
	if len(report.Components) != 2 {
		t.Fatalf("Components = %d, want 2", len(report.Components))
	}
	for _, c := range report.Components {
		if c.Status != "up" {
			t.Errorf("%s status = %q, want up", c.Name, c.Status)
		}
	}
}

func TestMonitor_Readiness_OneDown(t *testing.T) {
	m := NewMonitor()
	m.Register("db", CheckFunc(func(ctx context.Context) error { return nil }))
	m.Register("nats", CheckFunc(func(ctx context.Context) error {
		return errors.New("connection refused")
	}))

	report := m.Readiness(context.Background())
	if report.IsHealthy() {
		t.Error("Status = healthy, want unhealthy")
	}

	var natsStatus *Status
	for i := range report.Components {
		if report.Components[i].Name == "nats" {
			natsStatus = &report.Components[i]
		}
	}
	if natsStatus == nil {
		t.Fatal("nats component not found in report")
	}
	if natsStatus.Status != "down" {
		t.Errorf("nats status = %q, want down", natsStatus.Status)
	}
	if natsStatus.Error != "connection refused" {
		t.Errorf("nats error = %q, want 'connection refused'", natsStatus.Error)
	}
}

func TestMonitor_Readiness_Empty(t *testing.T) {
	m := NewMonitor()
	report := m.Readiness(context.Background())
	if !report.IsHealthy() {
		t.Error("empty monitor should be healthy")
	}
	if len(report.Components) != 0 {
		t.Errorf("Components = %d, want 0", len(report.Components))
	}
}

func TestMonitor_Liveness(t *testing.T) {
	m := NewMonitor()
	if err := m.Liveness(); err != nil {
		t.Errorf("Liveness() = %v, want nil", err)
	}
}

func TestMonitor_ReadinessError_Healthy(t *testing.T) {
	m := NewMonitor()
	m.Register("db", CheckFunc(func(ctx context.Context) error { return nil }))

	if err := m.ReadinessError(context.Background()); err != nil {
		t.Errorf("ReadinessError() = %v, want nil", err)
	}
}

func TestMonitor_ReadinessError_Unhealthy(t *testing.T) {
	m := NewMonitor()
	m.Register("db", CheckFunc(func(ctx context.Context) error {
		return errors.New("ping failed")
	}))

	err := m.ReadinessError(context.Background())
	if err == nil {
		t.Fatal("ReadinessError() = nil, want error")
	}
}

func TestCheckFunc_Satisfies_Checker(t *testing.T) {
	var c Checker = CheckFunc(func(ctx context.Context) error { return nil })
	if err := c.Check(context.Background()); err != nil {
		t.Errorf("Check() = %v, want nil", err)
	}
}

func TestMonitor_Readiness_PreservesOrder(t *testing.T) {
	m := NewMonitor()
	m.Register("alpha", CheckFunc(func(ctx context.Context) error { return nil }))
	m.Register("beta", CheckFunc(func(ctx context.Context) error { return nil }))
	m.Register("gamma", CheckFunc(func(ctx context.Context) error { return nil }))

	report := m.Readiness(context.Background())
	names := []string{"alpha", "beta", "gamma"}
	for i, want := range names {
		if report.Components[i].Name != want {
			t.Errorf("Components[%d].Name = %q, want %q", i, report.Components[i].Name, want)
		}
	}
}
