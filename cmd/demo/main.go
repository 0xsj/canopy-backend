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

	// ── 5. WebSocket hub ─────────────────────────────────────────
	hub := websocket.NewHub(wsCfg, log)
	upgrader := websocket.NewNhooyrUpgrader()

	// Bridge: NATS events → WebSocket broadcast.
	subscription, err := sub.Subscribe(ctx, "workspace.>", func(_ context.Context, event events.Event) error {
		log.Info("event received",
			logger.String("id", event.ID),
			logger.String("type", event.Type),
			logger.String("subject", event.Subject),
			logger.String("workspace", event.WorkspaceID),
		)

		// Forward to WebSocket clients in the same workspace.
		msg, err := websocket.NewMessage(event.Type, event.WorkspaceID, event.Data)
		if err != nil {
			return nil // log but don't fail the event
		}
		hub.BroadcastMessage(event.WorkspaceID, msg)
		return nil
	}, events.WithConsumer("demo_ws_bridge"))
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

	// ── 7. Routes ────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Health.
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

	// Leaves (simulated).
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
		page := types.PageResponse[map[string]any]{Items: leaves, HasMore: false}
		types.WriteOK(w, page)
	})

	// Publish a test event (now targets a specific workspace).
	mux.HandleFunc("POST /api/events/test", func(w http.ResponseWriter, r *http.Request) {
		wsID := r.URL.Query().Get("workspace_id")
		if wsID == "" {
			wsID = types.NewWorkspaceID().String()
		}

		event, err := events.New("leaf_created", wsID, map[string]string{
			"title":  "Test leaf from demo",
			"author": "demo",
		})
		if err != nil {
			types.WriteError(w, http.StatusInternalServerError, "event_build_failed", err.Error())
			return
		}
		event.Subject = events.BuildSubject(wsID, "exploration", "leaf_created")

		if err := pub.Publish(r.Context(), event); err != nil {
			types.WriteError(w, http.StatusInternalServerError, "event_publish_failed", err.Error())
			return
		}

		types.WriteOK(w, map[string]string{
			"event_id":     event.ID,
			"subject":      event.Subject,
			"workspace_id": wsID,
			"status":       "published",
		})
	})

	// Database info.
	mux.HandleFunc("GET /api/debug/db", func(w http.ResponseWriter, r *http.Request) {
		stats := db.Stats()
		httpserver.JSON(w, http.StatusOK, map[string]any{
			"total_conns":    stats.TotalConns,
			"idle_conns":     stats.IdleConns,
			"acquired_conns": stats.AcquiredConn,
		})
	})

	// WebSocket endpoint.
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		wsID := r.URL.Query().Get("workspace_id")
		if wsID == "" {
			http.Error(w, "workspace_id query parameter is required", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			log.Error("websocket upgrade failed", logger.Err(err))
			return
		}

		log.Info("websocket client connected",
			logger.String("workspace", wsID),
			logger.Int("room_size", hub.RoomSize(wsID)+1),
		)

		hub.ServeConn(r.Context(), conn, wsID, func(msg websocket.Message) {
			log.Debug("ws message from client",
				logger.String("type", msg.Type),
				logger.String("workspace", wsID),
			)
		})
	})

	// WebSocket stats.
	mux.HandleFunc("GET /api/debug/ws", func(w http.ResponseWriter, r *http.Request) {
		httpserver.JSON(w, http.StatusOK, map[string]any{
			"total_connections": hub.TotalConnections(),
		})
	})

	// ── 8. Middleware ────────────────────────────────────────────
	handler := httpserver.Chain(
		httpserver.Recovery(log),
		httpserver.RequestID(),
		httpserver.Logging(log),
	)(mux)

	// ── 9. Start server ──────────────────────────────────────────
	srv := httpserver.New(httpCfg, handler, log)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", logger.Err(err))
			stop()
		}
	}()

	log.Info("demo ready — endpoints:",
		logger.String("liveness", "GET  /healthz/live"),
		logger.String("readiness", "GET  /healthz/ready"),
		logger.String("leaves", "GET  /api/leaves"),
		logger.String("publish", "POST /api/events/test?workspace_id=ws_xxx"),
		logger.String("websocket", "WS   /ws?workspace_id=ws_xxx"),
		logger.String("ws_stats", "GET  /api/debug/ws"),
	)

	// ── 10. Graceful shutdown ────────────────────────────────────
	<-ctx.Done()
	log.Info("shutting down...")

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Error("server shutdown error", logger.Err(err))
	}

	log.Info("goodbye")
}
