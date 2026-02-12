package auth

import (
	"net/http"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/httpserver"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Middleware returns an HTTP middleware that validates bearer tokens.
// On success, decoded claims are stored in the request context (use FromClaims to retrieve).
// On failure, responds with 401 Unauthorized.
//
// Usage:
//
//	handler := httpserver.Chain(
//	    httpserver.Recovery(log),
//	    httpserver.RequestID(),
//	    auth.Middleware(validator, log),
//	)(mux)
func Middleware(v TokenValidator, log logger.Logger) httpserver.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := extractBearer(r)
			if !ok {
				httpserver.Error(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			claims, err := v.Validate(r.Context(), raw)
			if err != nil {
				log.Warn("auth: token validation failed",
					logger.String("method", r.Method),
					logger.String("path", r.URL.Path),
					logger.Err(err),
				)
				httpserver.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractBearer pulls the token from the Authorization header.
// Expects the format: "Bearer <token>".
func extractBearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return "", false
	}
	token := h[len("Bearer "):]
	if token == "" {
		return "", false
	}
	return token, true
}
