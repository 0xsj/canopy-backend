package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// testKeyPair holds an RSA key pair for test JWT signing.
type testKeyPair struct {
	private *rsa.PrivateKey
	kid     string
}

func newTestKeyPair(t *testing.T) testKeyPair {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return testKeyPair{private: key, kid: "test-key-1"}
}

// jwksJSON returns the JWKS document for the test key pair.
func (kp testKeyPair) jwksJSON() []byte {
	doc := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": kp.kid,
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(kp.private.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(kp.private.E)).Bytes()),
			},
		},
	}
	data, _ := json.Marshal(doc)
	return data
}

// serveJWKS starts a test server that serves the JWKS document.
func (kp testKeyPair) serveJWKS(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(kp.jwksJSON())
	}))
}

// signToken creates a signed JWT with the given claims.
func (kp testKeyPair) signToken(t *testing.T, mc jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, mc)
	token.Header["kid"] = kp.kid
	signed, err := token.SignedString(kp.private)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// validClaims returns a standard set of valid JWT claims for testing.
func validClaims(issuer, audience string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"sub":   "user_abc123",
		"email": "test@canopy.dev",
		"iss":   issuer,
		"aud":   audience,
		"exp":   now.Add(1 * time.Hour).Unix(),
		"iat":   now.Unix(),
	}
}

func TestJWKSValidator_Validate(t *testing.T) {
	kp := newTestKeyPair(t)
	server := kp.serveJWKS(t)
	defer server.Close()

	issuer := "https://test.auth0.com/"
	audience := "https://api.canopy.dev"

	v := NewJWKSValidator(Config{
		Issuer:   issuer,
		Audience: audience,
		JWKSURL:  server.URL,
	})

	rawToken := kp.signToken(t, validClaims(issuer, audience))

	claims, err := v.Validate(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.Subject != "user_abc123" {
		t.Errorf("Subject = %q, want user_abc123", claims.Subject)
	}
	if claims.Email != "test@canopy.dev" {
		t.Errorf("Email = %q, want test@canopy.dev", claims.Email)
	}
	if claims.Issuer != issuer {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, issuer)
	}
	if claims.Audience != audience {
		t.Errorf("Audience = %q, want %q", claims.Audience, audience)
	}
	if claims.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}
	if claims.IssuedAt.IsZero() {
		t.Error("IssuedAt should not be zero")
	}
}

func TestJWKSValidator_ExpiredToken(t *testing.T) {
	kp := newTestKeyPair(t)
	server := kp.serveJWKS(t)
	defer server.Close()

	issuer := "https://test.auth0.com/"
	audience := "https://api.canopy.dev"

	v := NewJWKSValidator(Config{
		Issuer:   issuer,
		Audience: audience,
		JWKSURL:  server.URL,
	})

	mc := jwt.MapClaims{
		"sub": "user_abc123",
		"iss": issuer,
		"aud": audience,
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	}
	rawToken := kp.signToken(t, mc)

	_, err := v.Validate(context.Background(), rawToken)
	if err == nil {
		t.Error("Validate() should reject expired token")
	}
}

func TestJWKSValidator_WrongAudience(t *testing.T) {
	kp := newTestKeyPair(t)
	server := kp.serveJWKS(t)
	defer server.Close()

	issuer := "https://test.auth0.com/"

	v := NewJWKSValidator(Config{
		Issuer:   issuer,
		Audience: "https://api.canopy.dev",
		JWKSURL:  server.URL,
	})

	mc := validClaims(issuer, "https://wrong-audience.com")
	rawToken := kp.signToken(t, mc)

	_, err := v.Validate(context.Background(), rawToken)
	if err == nil {
		t.Error("Validate() should reject wrong audience")
	}
}

func TestJWKSValidator_WrongIssuer(t *testing.T) {
	kp := newTestKeyPair(t)
	server := kp.serveJWKS(t)
	defer server.Close()

	audience := "https://api.canopy.dev"

	v := NewJWKSValidator(Config{
		Issuer:   "https://correct-issuer.auth0.com/",
		Audience: audience,
		JWKSURL:  server.URL,
	})

	mc := validClaims("https://wrong-issuer.auth0.com/", audience)
	rawToken := kp.signToken(t, mc)

	_, err := v.Validate(context.Background(), rawToken)
	if err == nil {
		t.Error("Validate() should reject wrong issuer")
	}
}

func TestJWKSValidator_UnknownKid(t *testing.T) {
	kp := newTestKeyPair(t)
	server := kp.serveJWKS(t)
	defer server.Close()

	issuer := "https://test.auth0.com/"
	audience := "https://api.canopy.dev"

	v := NewJWKSValidator(Config{
		Issuer:   issuer,
		Audience: audience,
		JWKSURL:  server.URL,
	})

	// Sign with a different kid that isn't in the JWKS.
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims(issuer, audience))
	token.Header["kid"] = "unknown-key-99"
	rawToken, err := token.SignedString(kp.private)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = v.Validate(context.Background(), rawToken)
	if err == nil {
		t.Error("Validate() should reject unknown kid")
	}
}

func TestJWKSValidator_InvalidSignature(t *testing.T) {
	kp := newTestKeyPair(t)
	server := kp.serveJWKS(t)
	defer server.Close()

	issuer := "https://test.auth0.com/"
	audience := "https://api.canopy.dev"

	v := NewJWKSValidator(Config{
		Issuer:   issuer,
		Audience: audience,
		JWKSURL:  server.URL,
	})

	// Sign with a different key than what's in the JWKS.
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims(issuer, audience))
	token.Header["kid"] = kp.kid // same kid, different key
	rawToken, _ := token.SignedString(otherKey)

	_, err := v.Validate(context.Background(), rawToken)
	if err == nil {
		t.Error("Validate() should reject token with invalid signature")
	}
}

func TestJWKSValidator_GarbageToken(t *testing.T) {
	kp := newTestKeyPair(t)
	server := kp.serveJWKS(t)
	defer server.Close()

	v := NewJWKSValidator(Config{
		Issuer:   "https://test.auth0.com/",
		Audience: "https://api.canopy.dev",
		JWKSURL:  server.URL,
	})

	_, err := v.Validate(context.Background(), "not.a.jwt")
	if err == nil {
		t.Error("Validate() should reject garbage input")
	}
}

func TestJWKSValidator_KeyCaching(t *testing.T) {
	kp := newTestKeyPair(t)

	fetchCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetchCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write(kp.jwksJSON())
	}))
	defer server.Close()

	issuer := "https://test.auth0.com/"
	audience := "https://api.canopy.dev"

	v := NewJWKSValidator(Config{
		Issuer:   issuer,
		Audience: audience,
		JWKSURL:  server.URL,
	})

	mc := validClaims(issuer, audience)

	// Validate twice — should only fetch JWKS once.
	for i := 0; i < 2; i++ {
		rawToken := kp.signToken(t, mc)
		if _, err := v.Validate(context.Background(), rawToken); err != nil {
			t.Fatalf("Validate() attempt %d error = %v", i+1, err)
		}
	}

	if fetchCount != 1 {
		t.Errorf("JWKS fetched %d times, want 1 (should be cached)", fetchCount)
	}
}

func TestMapToClaims(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	mc := jwt.MapClaims{
		"sub":   "user_xyz",
		"email": "xyz@canopy.dev",
		"iss":   "https://issuer.com/",
		"aud":   "https://api.example.com",
		"exp":   float64(now.Add(1 * time.Hour).Unix()),
		"iat":   float64(now.Unix()),
	}

	c := mapToClaims(mc)

	if c.Subject != "user_xyz" {
		t.Errorf("Subject = %q, want user_xyz", c.Subject)
	}
	if c.Email != "xyz@canopy.dev" {
		t.Errorf("Email = %q, want xyz@canopy.dev", c.Email)
	}
	if c.Issuer != "https://issuer.com/" {
		t.Errorf("Issuer = %q, want https://issuer.com/", c.Issuer)
	}
	if c.Audience != "https://api.example.com" {
		t.Errorf("Audience = %q, want https://api.example.com", c.Audience)
	}
}

func TestMapToClaims_MissingFields(t *testing.T) {
	mc := jwt.MapClaims{
		"sub": "user_only",
	}

	c := mapToClaims(mc)

	if c.Subject != "user_only" {
		t.Errorf("Subject = %q, want user_only", c.Subject)
	}
	if c.Email != "" {
		t.Errorf("Email = %q, want empty", c.Email)
	}
	if !c.ExpiresAt.IsZero() {
		t.Errorf("ExpiresAt = %v, want zero", c.ExpiresAt)
	}
}
