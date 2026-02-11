package events

import (
	"testing"
)

func TestBuildSubject(t *testing.T) {
	tests := []struct {
		workspace string
		context   string
		eventType string
		want      string
	}{
		{"ws_abc123", "exploration", "leaf_created", "workspace.ws_abc123.exploration.leaf_created"},
		{"ws_xyz", "convergence", "leaf_promoted", "workspace.ws_xyz.convergence.leaf_promoted"},
		{"*", "exploration", ">", "workspace.*.exploration.>"},
	}

	for _, tt := range tests {
		got := BuildSubject(tt.workspace, tt.context, tt.eventType)
		if got != tt.want {
			t.Errorf("BuildSubject(%q, %q, %q) = %q, want %q",
				tt.workspace, tt.context, tt.eventType, got, tt.want)
		}
	}
}

func TestParseSubject_Valid(t *testing.T) {
	ws, ctx, evt, err := ParseSubject("workspace.ws_abc123.exploration.leaf_created")
	if err != nil {
		t.Fatalf("ParseSubject() error: %v", err)
	}
	if ws != "ws_abc123" {
		t.Errorf("workspace = %q, want ws_abc123", ws)
	}
	if ctx != "exploration" {
		t.Errorf("context = %q, want exploration", ctx)
	}
	if evt != "leaf_created" {
		t.Errorf("eventType = %q, want leaf_created", evt)
	}
}

func TestParseSubject_Invalid(t *testing.T) {
	tests := []string{
		"",
		"workspace",
		"workspace.ws_abc",
		"workspace.ws_abc.exploration",
		"other.ws_abc.exploration.leaf_created",
		"invalid-subject",
	}

	for _, subject := range tests {
		_, _, _, err := ParseSubject(subject)
		if err == nil {
			t.Errorf("ParseSubject(%q) = nil, want error", subject)
		}
	}
}

func TestBuildSubject_ParseSubject_Roundtrip(t *testing.T) {
	ws, ctx, evt := "ws_roundtrip", "exploration", "leaf_created"
	subject := BuildSubject(ws, ctx, evt)

	gotWs, gotCtx, gotEvt, err := ParseSubject(subject)
	if err != nil {
		t.Fatalf("ParseSubject() error: %v", err)
	}
	if gotWs != ws {
		t.Errorf("workspace = %q, want %q", gotWs, ws)
	}
	if gotCtx != ctx {
		t.Errorf("context = %q, want %q", gotCtx, ctx)
	}
	if gotEvt != evt {
		t.Errorf("eventType = %q, want %q", gotEvt, evt)
	}
}
