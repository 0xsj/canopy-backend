package errors

import "testing"

func TestSentinels_HaveCorrectKind(t *testing.T) {
	tests := []struct {
		name     string
		sentinel *canopyError
		kind     Kind
	}{
		{"ErrNotFound", ErrNotFound, KindNotFound},
		{"ErrUnauthorized", ErrUnauthorized, KindAuthorization},
		{"ErrUnauthenticated", ErrUnauthenticated, KindAuthentication},
		{"ErrConflict", ErrConflict, KindConflict},
		{"ErrValidation", ErrValidation, KindValidation},
		{"ErrRateLimit", ErrRateLimit, KindRateLimit},
		{"ErrTimeout", ErrTimeout, KindTimeout},
		{"ErrInternal", ErrInternal, KindInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.sentinel.ErrorKind() != tt.kind {
				t.Errorf("%s kind = %v, want %v", tt.name, tt.sentinel.ErrorKind(), tt.kind)
			}
		})
	}
}

func TestSentinels_HaveCorrectSeverity(t *testing.T) {
	tests := []struct {
		name     string
		sentinel *canopyError
		severity Severity
	}{
		{"ErrNotFound", ErrNotFound, SeverityLow},
		{"ErrUnauthorized", ErrUnauthorized, SeverityMedium},
		{"ErrUnauthenticated", ErrUnauthenticated, SeverityMedium},
		{"ErrConflict", ErrConflict, SeverityLow},
		{"ErrValidation", ErrValidation, SeverityLow},
		{"ErrRateLimit", ErrRateLimit, SeverityMedium},
		{"ErrTimeout", ErrTimeout, SeverityHigh},
		{"ErrInternal", ErrInternal, SeverityHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.sentinel.ErrorSeverity() != tt.severity {
				t.Errorf("%s severity = %v, want %v", tt.name, tt.sentinel.ErrorSeverity(), tt.severity)
			}
		})
	}
}
