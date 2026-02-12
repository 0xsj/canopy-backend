package auth

import "context"

type claimsKey struct{}

// WithClaims stores Claims in the context.
// The auth middleware calls this after successful token validation.
func WithClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, c)
}

// FromClaims extracts Claims from the context.
// Returns zero Claims and false if none are present.
func FromClaims(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(Claims)
	return c, ok
}

// MustFromClaims extracts Claims from the context.
// Panics if no claims are present — use only in handlers behind auth middleware.
func MustFromClaims(ctx context.Context) Claims {
	c, ok := FromClaims(ctx)
	if !ok {
		panic("auth: no claims in context — handler not behind auth middleware")
	}
	return c
}
