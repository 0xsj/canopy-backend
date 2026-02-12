package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSValidator validates JWTs using public keys fetched from a JWKS endpoint.
// Works with any OIDC-compliant provider (Auth0, Clerk, etc.).
//
// Keys are cached in memory and refreshed when a token references an unknown
// key ID or the cache TTL expires.
type JWKSValidator struct {
	issuer   string
	audience string
	jwksURL  string
	client   *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	ttl       time.Duration
}

// NewJWKSValidator creates a production token validator.
// It fetches signing keys from the configured JWKS endpoint on demand.
func NewJWKSValidator(cfg Config) *JWKSValidator {
	return &JWKSValidator{
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		jwksURL:  cfg.JWKSURL,
		client:   &http.Client{Timeout: 10 * time.Second},
		keys:     make(map[string]*rsa.PublicKey),
		ttl:      1 * time.Hour,
	}
}

func (v *JWKSValidator) Validate(ctx context.Context, rawToken string) (Claims, error) {
	token, err := jwt.Parse(rawToken, v.keyFunc(ctx),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil {
		return Claims{}, fmt.Errorf("auth: validate token: %w", err)
	}

	mc, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, fmt.Errorf("auth: unexpected claims type")
	}

	return mapToClaims(mc), nil
}

// keyFunc returns a jwt.Keyfunc that resolves signing keys by key ID.
func (v *JWKSValidator) keyFunc(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("auth: missing kid header")
		}
		return v.getKey(ctx, kid)
	}
}

// getKey returns the RSA public key for the given key ID.
// Fetches fresh keys from the JWKS endpoint if needed.
func (v *JWKSValidator) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	stale := time.Since(v.fetchedAt) > v.ttl
	v.mu.RUnlock()

	if ok && !stale {
		return key, nil
	}

	if err := v.fetchKeys(ctx); err != nil {
		return nil, fmt.Errorf("auth: fetch jwks: %w", err)
	}

	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("auth: unknown key id: %s", kid)
	}
	return key, nil
}

// fetchKeys retrieves the JWKS document and parses RSA public keys.
func (v *JWKSValidator) fetchKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("auth: build jwks request: %w", err)
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth: jwks request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth: jwks endpoint returned %d", resp.StatusCode)
	}

	var doc struct {
		Keys []jwkEntry `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("auth: decode jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, entry := range doc.Keys {
		if entry.Kty != "RSA" || entry.Use != "sig" {
			continue
		}
		pub, err := entry.publicKey()
		if err != nil {
			continue // skip malformed keys
		}
		keys[entry.Kid] = pub
	}

	v.mu.Lock()
	v.keys = keys
	v.fetchedAt = time.Now()
	v.mu.Unlock()

	return nil
}

// jwkEntry represents a single key in the JWKS document.
type jwkEntry struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// publicKey converts the JWK entry into an RSA public key.
func (k jwkEntry) publicKey() (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("auth: decode modulus: %w", err)
	}
	eb, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("auth: decode exponent: %w", err)
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nb),
		E: int(new(big.Int).SetBytes(eb).Int64()),
	}, nil
}

// mapToClaims converts jwt.MapClaims into our Claims struct.
func mapToClaims(mc jwt.MapClaims) Claims {
	var c Claims

	if sub, err := mc.GetSubject(); err == nil {
		c.Subject = sub
	}
	if iss, err := mc.GetIssuer(); err == nil {
		c.Issuer = iss
	}
	if aud, err := mc.GetAudience(); err == nil && len(aud) > 0 {
		c.Audience = aud[0]
	}
	if email, ok := mc["email"].(string); ok {
		c.Email = email
	}
	if exp, err := mc.GetExpirationTime(); err == nil && exp != nil {
		c.ExpiresAt = exp.Time
	}
	if iat, err := mc.GetIssuedAt(); err == nil && iat != nil {
		c.IssuedAt = iat.Time
	}

	return c
}
