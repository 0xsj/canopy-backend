package auth

import "time"

// Claims represents the decoded payload from a validated JWT.
// These are standard OIDC claims extracted after token verification.
// Claims flow through context.Context to downstream handlers and services.
type Claims struct {
	Subject   string    // sub — provider's unique user identifier
	Email     string    // email
	Issuer    string    // iss — token issuer URL
	Audience  string    // aud — intended recipient
	ExpiresAt time.Time // exp
	IssuedAt  time.Time // iat
}

// IsExpired reports whether the token has expired.
func (c Claims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsZero reports whether the claims are empty (no subject set).
func (c Claims) IsZero() bool {
	return c.Subject == ""
}
