package auth

import "context"

// TokenValidator validates a raw bearer token and returns decoded claims.
// Implementations handle signature verification, expiry checks, and issuer validation.
//
// This is the port. Adapters include:
//   - JWKSValidator: production validator using JWKS key fetching (Auth0, Clerk, any OIDC provider)
//   - StaticValidator: returns fixed claims for tests and local development
type TokenValidator interface {
	Validate(ctx context.Context, token string) (Claims, error)
}
