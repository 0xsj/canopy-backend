package errors

// Kind classifies what category an error belongs to.
// Handlers use kind to determine HTTP status codes and client responses.
type Kind int

const (
	KindUnknown Kind = iota
	KindValidation
	KindNotFound
	KindConflict
	KindAuthorization
	KindAuthentication
	KindRateLimit
	KindTimeout
	KindInternal
)

var kindStrings = map[Kind]string{
	KindUnknown:        "unknown",
	KindValidation:     "validation",
	KindNotFound:       "not_found",
	KindConflict:       "conflict",
	KindAuthorization:  "authorization",
	KindAuthentication: "authentication",
	KindRateLimit:      "rate_limit",
	KindTimeout:        "timeout",
	KindInternal:       "internal",
}

func (k Kind) String() string {
	if str, ok := kindStrings[k]; ok {
		return str
	}
	return "unknown"
}
