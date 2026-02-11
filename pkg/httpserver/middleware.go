package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Middleware is the standard middleware signature.
type Middleware func(http.Handler) http.Handler

// Chain composes middlewares left-to-right. The first middleware
// in the list is the outermost (runs first on request, last on response).
//
// Usage:
//
//	handler := httpserver.Chain(
//	    httpserver.Recovery(log),
//	    httpserver.RequestID(),
//	    httpserver.Logging(log),
//	)(mux)
func Chain(mws ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

// Recovery catches panics in downstream handlers, logs the error,
// and returns a 500 response instead of crashing the server.
func Recovery(log logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log.Error("panic recovered",
						logger.String("method", r.Method),
						logger.String("path", r.URL.Path),
						logger.Any("panic", v),
					)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID generates a unique ID for each request and sets it
// on the response header and request context.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				b := make([]byte, 8)
				rand.Read(b)
				id = hex.EncodeToString(b)
			}
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r)
		})
	}
}

// Logging logs each request with method, path, status, and duration.
func Logging(log logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			log.Info("http request",
				logger.String("method", r.Method),
				logger.String("path", r.URL.Path),
				logger.Int("status", sw.status),
				logger.Any("duration", time.Since(start)),
			)
		})
	}
}

// statusWriter wraps ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
