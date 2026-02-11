package errors

import "testing"

func TestKind_String(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindUnknown, "unknown"},
		{KindValidation, "validation"},
		{KindNotFound, "not_found"},
		{KindConflict, "conflict"},
		{KindAuthorization, "authorization"},
		{KindAuthentication, "authentication"},
		{KindRateLimit, "rate_limit"},
		{KindTimeout, "timeout"},
		{KindInternal, "internal"},
		{Kind(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("Kind.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
