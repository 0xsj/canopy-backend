package httpserver

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig controls Cross-Origin Resource Sharing headers.
type CORSConfig struct {
	AllowedOrigins []string // ["*"] allows all; otherwise match request Origin
	AllowedMethods []string // default: GET, POST, PATCH, PUT, DELETE, OPTIONS
	AllowedHeaders []string // default: Authorization, Content-Type, X-Request-ID
	MaxAge         int      // preflight cache seconds; default: 3600
}

func (c *CORSConfig) defaults() {
	if len(c.AllowedMethods) == 0 {
		c.AllowedMethods = []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"}
	}
	if len(c.AllowedHeaders) == 0 {
		c.AllowedHeaders = []string{"Authorization", "Content-Type", "X-Request-ID"}
	}
	if c.MaxAge == 0 {
		c.MaxAge = 3600
	}
}

// CORS returns middleware that sets CORS response headers and handles
// preflight OPTIONS requests with a 204 short-circuit.
func CORS(cfg CORSConfig) Middleware {
	cfg.defaults()

	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	allowAll := len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*"
	origins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		origins[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				if _, ok := origins[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			w.Header().Set("Access-Control-Max-Age", maxAge)

			// Preflight — short-circuit with 204.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
