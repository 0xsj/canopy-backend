# 2026-02-12 — Auth Package

## What was done
- Built `pkg/auth` — token validation port, claims, middleware, and two validator implementations
- `claims.go` — Claims struct with standard OIDC fields (Subject, Email, Issuer, Audience, ExpiresAt, IssuedAt)
- `validator.go` — TokenValidator interface (the port)
- `context.go` — WithClaims / FromClaims / MustFromClaims context helpers
- `config.go` — Config section (ISSUER, AUDIENCE, JWKS_URL) following existing section pattern
- `middleware.go` — HTTP middleware: extracts Bearer token, validates, injects claims into ctx
- `static.go` — StaticValidator (fixed claims) and FailingValidator (fixed error) for tests/dev
- `jwks.go` — JWKSValidator: JWKS key fetching, RSA public key parsing, JWT verification with caching
- Full test suite: 25 tests across 6 test files covering all components including JWKS with test key generation
- `cmd/authdemo` — standalone demo with browser UI, public/protected endpoints, curl instructions

## Decisions made
- Chose `golang-jwt/jwt/v5` as the JWT library — most widely used, gives full control over JWKS handling
- Hand-rolled JWKS fetcher rather than using go-oidc — fewer dependencies, works with any OIDC provider
- Auth package is shared infrastructure (`pkg/auth`), not a bounded context — authorization stays in domain layer
- JWKS keys cached in memory with 1-hour TTL, automatic refresh on unknown kid

## Open questions
- None for this package — ready for use by bounded contexts

## Next steps
- Add CORS + security headers middleware to `pkg/httpserver`
- Begin domain types for the first bounded context (identity or workspace)
- Wire `pkg/auth` into the composition root (`cmd/server/`)
