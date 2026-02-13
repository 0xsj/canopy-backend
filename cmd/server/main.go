package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	// internal: bounded context providers
	"github.com/0xsj/canopy-backend/internal/convergence"
	"github.com/0xsj/canopy-backend/internal/deliverable"
	"github.com/0xsj/canopy-backend/internal/discussion"
	"github.com/0xsj/canopy-backend/internal/exploration"
	"github.com/0xsj/canopy-backend/internal/identity"
	"github.com/0xsj/canopy-backend/internal/ledger"
	"github.com/0xsj/canopy-backend/internal/notification"
	"github.com/0xsj/canopy-backend/internal/organization"
	"github.com/0xsj/canopy-backend/internal/seed"
	"github.com/0xsj/canopy-backend/internal/session"
	"github.com/0xsj/canopy-backend/internal/synthesis"
	"github.com/0xsj/canopy-backend/internal/workspace"

	// pkg
	"github.com/0xsj/canopy-backend/pkg/auth"
	"github.com/0xsj/canopy-backend/pkg/config"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/health"
	"github.com/0xsj/canopy-backend/pkg/httpserver"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/websocket"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── 1. Config ────────────────────────────────────────────────
	var httpCfg httpserver.Config
	var dbCfg database.Config
	var eventsCfg events.Config
	var wsCfg websocket.Config

	loader := config.NewLoader("CANOPY")
	loader.Register("HTTP", &httpCfg)
	loader.Register("DATABASE", &dbCfg)
	loader.Register("EVENTS", &eventsCfg)
	loader.Register("WS", &wsCfg)

	if err := loader.LoadAll(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}

	// ── 2. Logger ────────────────────────────────────────────────
	log := logger.NewConsole(
		logger.WithLevel(logger.LevelDebug),
		logger.WithColor(true),
		logger.WithTimestamps(true),
	)

	// ── 3. Database ──────────────────────────────────────────────
	db, err := database.Connect(ctx, dbCfg, log)
	if err != nil {
		log.Error("database connection failed", logger.Err(err))
		os.Exit(1)
	}
	defer db.Close()

	// ── 4. Events (NATS + JetStream) ─────────────────────────────
	broker, err := events.Connect(ctx, eventsCfg, log)
	if err != nil {
		log.Error("nats connection failed", logger.Err(err))
		os.Exit(1)
	}
	defer broker.Close()

	pub := events.NewPublisher(broker)
	sub := events.NewSubscriber(broker)

	// ── 5. WebSocket hub + NATS→WS bridge ────────────────────────
	hub := websocket.NewHub(wsCfg, log)
	upgrader := websocket.NewNhooyrUpgrader()

	subscription, err := sub.Subscribe(ctx, "workspace.>", func(_ context.Context, event events.Event) error {
		msg, err := websocket.NewMessage(event.Type, event.WorkspaceID, event.Data)
		if err != nil {
			log.Warn("ws bridge: failed to build message", logger.Err(err))
			return nil
		}
		hub.BroadcastMessage(event.WorkspaceID, msg)
		return nil
	}, events.WithConsumer("server_ws_bridge"))
	if err != nil {
		log.Error("event subscription failed", logger.Err(err))
		os.Exit(1)
	}
	defer subscription.Unsubscribe()

	// ── 6. Health monitor ────────────────────────────────────────
	monitor := health.NewMonitor()
	monitor.Register("database", health.CheckFunc(db.Health))
	monitor.Register("nats", health.CheckFunc(func(ctx context.Context) error {
		return broker.Health()
	}))

	// ── 7. Providers ────────────────────────────────────────────
	identityP := identity.Wire(db, pub, log)
	ledgerP := ledger.Wire(db, log)
	notifP := notification.Wire(db, nil, pub, log) // no transports yet

	orgP := organization.Wire(db, &userReaderAdapter{repo: identityP.UserRepo}, pub, log)
	wsP := workspace.Wire(db, orgP.MemberRepo, pub, log)

	seedP := seed.Wire(db, wsP.MemberRepo, pub, log)
	expP := exploration.Wire(db, wsP.MemberRepo, pub, log)
	discP := discussion.Wire(db, wsP.MemberRepo, pub, log)
	sessP := session.Wire(db, noopContextAssembler{}, wsP.MemberRepo, pub, log)
	synthP := synthesis.Wire(db, wsP.MemberRepo, pub, log)
	delP := deliverable.Wire(db, wsP.MemberRepo, pub, log)
	convP := convergence.Wire(db, &leafWriterAdapter{repo: expP.LeafRepo}, wsP.MemberRepo, pub, log)

	// ── 7b. Event subscribers ───────────────────────────────────
	cleanupSubscribers, err := registerSubscribers(ctx, sub, ledgerP.Service, log)
	if err != nil {
		log.Error("subscriber registration failed", logger.Err(err))
		os.Exit(1)
	}
	defer cleanupSubscribers()

	// ── 8. Routes ───────────────────────────────────────────────
	mux := http.NewServeMux()

	// Health — unauthenticated.
	mux.HandleFunc("GET /healthz/live", func(w http.ResponseWriter, r *http.Request) {
		httpserver.JSON(w, http.StatusOK, map[string]string{"status": "alive"})
	})
	mux.HandleFunc("GET /healthz/ready", func(w http.ResponseWriter, r *http.Request) {
		report := monitor.Readiness(r.Context())
		status := http.StatusOK
		if !report.IsHealthy() {
			status = http.StatusServiceUnavailable
		}
		httpserver.JSON(w, status, report)
	})

	// WebSocket — own auth (not behind API middleware).
	mux.Handle("GET /ws", websocket.Handler(hub, upgrader, log))

	// API — all routes behind auth middleware.
	validator := auth.NewStaticValidator(auth.Claims{
		Subject: "dev_user",
		Issuer:  "canopy-dev",
	})

	apiMux := http.NewServeMux()
	identityP.Handler.Register(apiMux)
	orgP.Handler.Register(apiMux)
	wsP.Handler.Register(apiMux)
	seedP.Handler.Register(apiMux)
	expP.Handler.Register(apiMux)
	discP.Handler.Register(apiMux)
	convP.Handler.Register(apiMux)
	sessP.Handler.Register(apiMux)
	synthP.Handler.Register(apiMux)
	delP.Handler.Register(apiMux)
	notifP.Handler.Register(apiMux)
	ledgerP.Handler.Register(apiMux)

	mux.Handle("/api/", auth.Middleware(validator, log)(apiMux))

	// ── 9. Middleware ───────────────────────────────────────────
	handler := httpserver.Chain(
		httpserver.Recovery(log),
		httpserver.RequestID(),
		httpserver.CORS(httpserver.CORSConfig{
			AllowedOrigins: []string{"*"},
		}),
		httpserver.Logging(log),
	)(mux)

	// ── 10. Start server + graceful shutdown ─────────────────────
	srv := httpserver.New(httpCfg, handler, log)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", logger.Err(err))
			stop()
		}
	}()

	log.Info("canopy server ready",
		logger.String("addr", fmt.Sprintf(":%d", httpCfg.Port)),
		logger.String("auth", "static (dev_user)"),
	)

	<-ctx.Done()
	log.Info("shutting down...")

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Error("server shutdown error", logger.Err(err))
	}

	log.Info("goodbye")
}
