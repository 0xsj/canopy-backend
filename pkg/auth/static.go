package auth

import "context"

// StaticValidator returns fixed claims for every token.
// Used in unit tests and local development where no OAuth provider is running.
//
// Usage:
//
//	v := auth.NewStaticValidator(auth.Claims{
//	    Subject: "user_test123",
//	    Email:   "dev@canopy.dev",
//	})
//	middleware := auth.Middleware(v, log)
type StaticValidator struct {
	claims Claims
	err    error
}

// NewStaticValidator creates a validator that always returns the given claims.
func NewStaticValidator(claims Claims) *StaticValidator {
	return &StaticValidator{claims: claims}
}

// NewFailingValidator creates a validator that always returns the given error.
// Useful for testing error paths in handlers.
func NewFailingValidator(err error) *StaticValidator {
	return &StaticValidator{err: err}
}

func (v *StaticValidator) Validate(_ context.Context, _ string) (Claims, error) {
	if v.err != nil {
		return Claims{}, v.err
	}
	return v.claims, nil
}
