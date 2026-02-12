package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/0xsj/canopy-backend/pkg/auth"
	"github.com/0xsj/canopy-backend/pkg/httpserver"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

const testToken = "dev-token-canopy-2026"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log := logger.NewConsole(
		logger.WithLevel(logger.LevelDebug),
		logger.WithColor(true),
		logger.WithTimestamps(true),
	)

	// Static validator for local development — accepts any token
	// and returns these fixed claims.
	validator := auth.NewStaticValidator(auth.Claims{
		Subject:   "user_dev_abc123",
		Email:     "dev@canopy.dev",
		Issuer:    "static-dev",
		Audience:  "canopy-authdemo",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IssuedAt:  time.Now(),
	})

	mux := http.NewServeMux()

	// Public endpoint — no auth required.
	mux.HandleFunc("GET /public", func(w http.ResponseWriter, r *http.Request) {
		httpserver.JSON(w, http.StatusOK, map[string]string{
			"message": "this endpoint is public — no token required",
		})
	})

	// Protected endpoint — requires valid bearer token.
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("GET /me", func(w http.ResponseWriter, r *http.Request) {
		claims, _ := auth.FromClaims(r.Context())
		httpserver.JSON(w, http.StatusOK, map[string]any{
			"message":  "authenticated",
			"subject":  claims.Subject,
			"email":    claims.Email,
			"issuer":   claims.Issuer,
			"audience": claims.Audience,
			"expired":  claims.IsExpired(),
		})
	})

	// Mount protected routes behind auth middleware.
	mux.Handle("/api/", http.StripPrefix("/api",
		auth.Middleware(validator, log)(protectedMux),
	))

	// Browser UI.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, indexHTML)
	})

	handler := httpserver.Chain(
		httpserver.Recovery(log),
		httpserver.RequestID(),
		httpserver.Logging(log),
	)(mux)

	srv := httpserver.New(httpserver.Config{
		Host: "0.0.0.0",
		Port: 8082,
	}, handler, log)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", logger.Err(err))
			stop()
		}
	}()

	log.Info("authdemo ready",
		logger.String("addr", "http://localhost:8082"),
	)
	log.Info("test commands:")
	log.Info("  curl http://localhost:8082/public")
	log.Info("  curl http://localhost:8082/api/me")
	log.Info("  curl -H 'Authorization: Bearer " + testToken + "' http://localhost:8082/api/me")

	<-ctx.Done()
	log.Info("shutting down...")
	srv.Shutdown(context.Background())
	log.Info("goodbye")
}

const indexHTML = `<!DOCTYPE html>
<html>
<head>
<title>Canopy Auth Demo</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, system-ui, sans-serif; background: #0d1117; color: #c9d1d9; padding: 24px; }
  h1 { color: #58a6ff; margin-bottom: 8px; font-size: 20px; }
  .subtitle { color: #8b949e; margin-bottom: 24px; font-size: 14px; }
  .grid { display: grid; grid-template-columns: 320px 1fr; gap: 24px; max-width: 900px; }
  .panel { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 16px; }
  .panel h2 { font-size: 14px; color: #8b949e; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 12px; }
  label { display: block; font-size: 13px; color: #8b949e; margin-bottom: 4px; }
  input { width: 100%; padding: 8px 10px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 14px; margin-bottom: 12px; font-family: 'SF Mono', monospace; }
  input:focus { outline: none; border-color: #58a6ff; }
  button { padding: 8px 16px; border: none; border-radius: 6px; font-size: 13px; cursor: pointer; font-weight: 600; width: 100%; margin-bottom: 8px; }
  .btn-public { background: #238636; color: white; }
  .btn-public:hover { background: #2ea043; }
  .btn-protected { background: #1f6feb; color: white; }
  .btn-protected:hover { background: #388bfd; }
  .btn-noauth { background: #da3633; color: white; }
  .btn-noauth:hover { background: #f85149; }
  .result { min-height: 300px; font-family: 'SF Mono', monospace; font-size: 13px; line-height: 1.6; padding: 12px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; white-space: pre-wrap; overflow-y: auto; }
  .status { font-size: 13px; margin-bottom: 8px; padding: 6px 10px; border-radius: 4px; }
  .status.ok { background: #0d2818; color: #3fb950; border: 1px solid #238636; }
  .status.err { background: #2d1117; color: #f85149; border: 1px solid #da3633; }
  .info { font-size: 12px; color: #484f58; margin-top: 8px; }
</style>
</head>
<body>

<h1>Canopy Auth Demo</h1>
<p class="subtitle">Test authentication middleware with public and protected endpoints</p>

<div class="grid">
  <div>
    <div class="panel">
      <h2>Token</h2>
      <label>Bearer Token</label>
      <input id="token" value="` + testToken + `" placeholder="paste your bearer token..." />
      <p class="info">The static validator accepts any token value for local development.</p>
    </div>

    <div class="panel" style="margin-top: 16px;">
      <h2>Endpoints</h2>
      <button class="btn-public" onclick="fetchEndpoint('/public', false)">
        GET /public (no auth)
      </button>
      <button class="btn-protected" onclick="fetchEndpoint('/api/me', true)">
        GET /api/me (with token)
      </button>
      <button class="btn-noauth" onclick="fetchEndpoint('/api/me', false)">
        GET /api/me (no token)
      </button>
    </div>

    <div class="panel" style="margin-top: 16px;">
      <h2>How It Works</h2>
      <p class="info">
        1. /public — no middleware, always 200<br><br>
        2. /api/* — behind auth.Middleware<br><br>
        3. Middleware extracts Bearer token from Authorization header<br><br>
        4. Validates via TokenValidator (StaticValidator in dev)<br><br>
        5. Injects Claims into context<br><br>
        6. Handler reads claims via auth.FromClaims(ctx)
      </p>
    </div>
  </div>

  <div>
    <div class="panel">
      <h2>Response</h2>
      <div id="status"></div>
      <div class="result" id="result">Click an endpoint to see the response...</div>
    </div>
  </div>
</div>

<script>
async function fetchEndpoint(path, withAuth) {
  const headers = {};
  if (withAuth) {
    const token = document.getElementById('token').value.trim();
    if (token) headers['Authorization'] = 'Bearer ' + token;
  }

  try {
    const resp = await fetch(path, { headers });
    const body = await resp.json();
    const statusEl = document.getElementById('status');
    const resultEl = document.getElementById('result');

    statusEl.className = 'status ' + (resp.ok ? 'ok' : 'err');
    statusEl.textContent = resp.status + ' ' + resp.statusText + '  ' + (withAuth ? '(with token)' : '(no token)');
    resultEl.textContent = JSON.stringify(body, null, 2);
  } catch(e) {
    document.getElementById('status').className = 'status err';
    document.getElementById('status').textContent = 'Error';
    document.getElementById('result').textContent = e.message;
  }
}
</script>
</body>
</html>`
