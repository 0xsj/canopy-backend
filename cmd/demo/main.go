package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/0xsj/canopy-backend/pkg/config"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/health"
	"github.com/0xsj/canopy-backend/pkg/httpserver"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── 1. Config ────────────────────────────────────────────────
	var httpCfg httpserver.Config
	var dbCfg database.Config
	var eventsCfg events.Config

	loader := config.NewLoader("CANOPY")
	loader.Register("HTTP", &httpCfg)
	loader.Register("DATABASE", &dbCfg)
	loader.Register("EVENTS", &eventsCfg)

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

	// Subscribe to all workspace events — demo event logger.
	subscription, err := sub.Subscribe(ctx, "workspace.>", func(_ context.Context, event events.Event) error {
		log.Info("event received",
			logger.String("id", event.ID),
			logger.String("type", event.Type),
			logger.String("subject", event.Subject),
			logger.String("workspace", event.WorkspaceID),
		)
		return nil
	}, events.WithConsumer("demo_logger"))
	if err != nil {
		log.Error("event subscription failed", logger.Err(err))
		os.Exit(1)
	}
	defer subscription.Unsubscribe()

	// ── 5. Health monitor ────────────────────────────────────────
	monitor := health.NewMonitor()
	monitor.Register("database", health.CheckFunc(db.Health))
	monitor.Register("nats", health.CheckFunc(func(ctx context.Context) error {
		return broker.Health()
	}))

	// ── 6. Routes ────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Health endpoints.
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

	// Simulated leaves endpoint.
	mux.HandleFunc("GET /api/leaves", func(w http.ResponseWriter, r *http.Request) {
		wsID := types.NewWorkspaceID()
		leaves := make([]map[string]any, 3)
		for i := range 3 {
			leaves[i] = map[string]any{
				"id":           types.NewLeafID().String(),
				"title":        fmt.Sprintf("Leaf #%d", i+1),
				"workspace_id": wsID.String(),
				"created_at":   types.Now().String(),
			}
		}

		page := types.PageResponse[map[string]any]{
			Items:   leaves,
			HasMore: false,
		}
		types.WriteOK(w, page)
	})

	// Publish a test event.
	mux.HandleFunc("POST /api/events/test", func(w http.ResponseWriter, r *http.Request) {
		wsID := types.NewWorkspaceID()

		event, err := events.New("leaf_created", wsID.String(), map[string]string{
			"title":  "Test leaf from demo",
			"author": "demo",
		})
		if err != nil {
			types.WriteError(w, http.StatusInternalServerError, "event_build_failed", err.Error())
			return
		}
		event.Subject = events.BuildSubject(wsID.String(), "exploration", "leaf_created")

		if err := pub.Publish(r.Context(), event); err != nil {
			types.WriteError(w, http.StatusInternalServerError, "event_publish_failed", err.Error())
			return
		}

		types.WriteOK(w, map[string]string{
			"event_id": event.ID,
			"subject":  event.Subject,
			"status":   "published",
		})
	})

	// Database info endpoint.
	mux.HandleFunc("GET /api/debug/db", func(w http.ResponseWriter, r *http.Request) {
		stats := db.Stats()
		httpserver.JSON(w, http.StatusOK, map[string]any{
			"total_conns":   stats.TotalConns,
			"idle_conns":    stats.IdleConns,
			"acquired_conns": stats.AcquiredConn,
		})
	})

	// ── 7. Middleware ────────────────────────────────────────────
	handler := httpserver.Chain(
		httpserver.Recovery(log),
		httpserver.RequestID(),
		httpserver.Logging(log),
	)(mux)

	// ── 8. Start server ──────────────────────────────────────────
	srv := httpserver.New(httpCfg, handler, log)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", logger.Err(err))
			stop()
		}
	}()

	log.Info("demo ready — try these endpoints:",
		logger.String("liveness", "GET http://localhost:8080/healthz/live"),
		logger.String("readiness", "GET http://localhost:8080/healthz/ready"),
		logger.String("leaves", "GET http://localhost:8080/api/leaves"),
		logger.String("publish", "POST http://localhost:8080/api/events/test"),
		logger.String("db_stats", "GET http://localhost:8080/api/debug/db"),
	)

	// ── 9. Graceful shutdown ─────────────────────────────────────
	<-ctx.Done()
	log.Info("shutting down...")

	shutdownCtx := context.Background()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", logger.Err(err))
	}

	log.Info("goodbye")
}
