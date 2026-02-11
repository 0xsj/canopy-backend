package types

import "testing"

func TestPageRequest_EffectiveLimit_Default(t *testing.T) {
	p := PageRequest{}
	if got := p.EffectiveLimit(); got != DefaultPageSize {
		t.Errorf("EffectiveLimit() = %d, want %d", got, DefaultPageSize)
	}
}

func TestPageRequest_EffectiveLimit_Negative(t *testing.T) {
	p := PageRequest{Limit: -5}
	if got := p.EffectiveLimit(); got != DefaultPageSize {
		t.Errorf("EffectiveLimit(-5) = %d, want %d", got, DefaultPageSize)
	}
}

func TestPageRequest_EffectiveLimit_Clamped(t *testing.T) {
	p := PageRequest{Limit: 500}
	if got := p.EffectiveLimit(); got != MaxPageSize {
		t.Errorf("EffectiveLimit(500) = %d, want %d", got, MaxPageSize)
	}
}

func TestPageRequest_EffectiveLimit_Valid(t *testing.T) {
	p := PageRequest{Limit: 10}
	if got := p.EffectiveLimit(); got != 10 {
		t.Errorf("EffectiveLimit(10) = %d, want 10", got)
	}
}

func TestPageRequest_EffectiveDirection_Default(t *testing.T) {
	p := PageRequest{}
	if got := p.EffectiveDirection(); got != DirectionForward {
		t.Errorf("EffectiveDirection() = %q, want %q", got, DirectionForward)
	}
}

func TestPageRequest_EffectiveDirection_Backward(t *testing.T) {
	p := PageRequest{Direction: DirectionBackward}
	if got := p.EffectiveDirection(); got != DirectionBackward {
		t.Errorf("EffectiveDirection() = %q, want %q", got, DirectionBackward)
	}
}

func TestNewPageResponse_HasMore(t *testing.T) {
	items := []string{"a", "b", "c", "d"}
	limit := 3

	resp := NewPageResponse(items, limit, func(s string) string { return s })

	if len(resp.Items) != 3 {
		t.Errorf("Items count = %d, want 3", len(resp.Items))
	}
	if !resp.HasMore {
		t.Error("HasMore = false, want true")
	}
	if resp.NextCursor != "c" {
		t.Errorf("NextCursor = %q, want %q", resp.NextCursor, "c")
	}
}

func TestNewPageResponse_NoMore(t *testing.T) {
	items := []string{"a", "b"}
	limit := 3

	resp := NewPageResponse(items, limit, func(s string) string { return s })

	if len(resp.Items) != 2 {
		t.Errorf("Items count = %d, want 2", len(resp.Items))
	}
	if resp.HasMore {
		t.Error("HasMore = true, want false")
	}
	if resp.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty", resp.NextCursor)
	}
}

func TestNewPageResponse_ExactLimit(t *testing.T) {
	items := []string{"a", "b", "c"}
	limit := 3

	resp := NewPageResponse(items, limit, func(s string) string { return s })

	if len(resp.Items) != 3 {
		t.Errorf("Items count = %d, want 3", len(resp.Items))
	}
	if resp.HasMore {
		t.Error("HasMore = true, want false when items == limit")
	}
}

func TestNewPageResponse_Empty(t *testing.T) {
	var items []string
	resp := NewPageResponse(items, 10, func(s string) string { return s })

	if len(resp.Items) != 0 {
		t.Errorf("Items count = %d, want 0", len(resp.Items))
	}
	if resp.HasMore {
		t.Error("HasMore = true, want false for empty")
	}
}

func TestCursor_RoundTrip(t *testing.T) {
	raw := "leaf_abc123_2026-02-11T15:30:00Z"
	encoded := EncodeCursor(raw)
	decoded := DecodeCursor(encoded)

	if decoded != raw {
		t.Errorf("cursor round-trip: got %q, want %q", decoded, raw)
	}
}

func TestCursor_OpaqueFormat(t *testing.T) {
	encoded := EncodeCursor("leaf_abc123")
	// Should not contain the raw value directly.
	if encoded == "leaf_abc123" {
		t.Error("encoded cursor should be opaque, not raw")
	}
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	got := DecodeCursor("!!!not-base64!!!")
	if got != "" {
		t.Errorf("invalid cursor decode = %q, want empty", got)
	}
}
