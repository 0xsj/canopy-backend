package errors

// Sentinel errors are predefined, well-known errors that callers
// check against using errors.Is. Each bounded context wraps these
// with its own operation and metadata — the sentinel identifies the category.

var (
	ErrNotFound        = New("not found").WithKind(KindNotFound).WithSeverity(SeverityLow)
	ErrUnauthorized    = New("unauthorized").WithKind(KindAuthorization).WithSeverity(SeverityMedium)
	ErrUnauthenticated = New("unauthenticated").WithKind(KindAuthentication).WithSeverity(SeverityMedium)
	ErrConflict        = New("conflict").WithKind(KindConflict).WithSeverity(SeverityLow)
	ErrValidation      = New("validation failed").WithKind(KindValidation).WithSeverity(SeverityLow)
	ErrRateLimit       = New("rate limit exceeded").WithKind(KindRateLimit).WithSeverity(SeverityMedium)
	ErrTimeout         = New("operation timed out").WithKind(KindTimeout).WithSeverity(SeverityHigh)
	ErrInternal        = New("internal error").WithKind(KindInternal).WithSeverity(SeverityHigh)
)
