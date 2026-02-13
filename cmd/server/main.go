package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	// internal: domain (tx factory return types)
	expdomain "github.com/0xsj/canopy-backend/internal/exploration/domain"
	orgdomain "github.com/0xsj/canopy-backend/internal/organization/domain"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"

	// internal: adapters (postgres)
	convpg "github.com/0xsj/canopy-backend/internal/convergence/adapter/postgres"
	delpg "github.com/0xsj/canopy-backend/internal/deliverable/adapter/postgres"
	discpg "github.com/0xsj/canopy-backend/internal/discussion/adapter/postgres"
	exppg "github.com/0xsj/canopy-backend/internal/exploration/adapter/postgres"
	identitypg "github.com/0xsj/canopy-backend/internal/identity/adapter/postgres"
	ledgerpg "github.com/0xsj/canopy-backend/internal/ledger/adapter/postgres"
	notifpg "github.com/0xsj/canopy-backend/internal/notification/adapter/postgres"
	orgpg "github.com/0xsj/canopy-backend/internal/organization/adapter/postgres"
	seedpg "github.com/0xsj/canopy-backend/internal/seed/adapter/postgres"
	sesspg "github.com/0xsj/canopy-backend/internal/session/adapter/postgres"
	synthpg "github.com/0xsj/canopy-backend/internal/synthesis/adapter/postgres"
	wspg "github.com/0xsj/canopy-backend/internal/workspace/adapter/postgres"

	// internal: services
	convsvc "github.com/0xsj/canopy-backend/internal/convergence/service"
	delsvc "github.com/0xsj/canopy-backend/internal/deliverable/service"
	discsvc "github.com/0xsj/canopy-backend/internal/discussion/service"
	expsvc "github.com/0xsj/canopy-backend/internal/exploration/service"
	identitysvc "github.com/0xsj/canopy-backend/internal/identity/service"
	ledgersvc "github.com/0xsj/canopy-backend/internal/ledger/service"
	notifsvc "github.com/0xsj/canopy-backend/internal/notification/service"
	orgsvc "github.com/0xsj/canopy-backend/internal/organization/service"
	seedsvc "github.com/0xsj/canopy-backend/internal/seed/service"
	sesssvc "github.com/0xsj/canopy-backend/internal/session/service"
	synthsvc "github.com/0xsj/canopy-backend/internal/synthesis/service"
	wssvc "github.com/0xsj/canopy-backend/internal/workspace/service"

	// internal: HTTP handlers
	convhttp "github.com/0xsj/canopy-backend/internal/convergence/interface/http/v1"
	delhttp "github.com/0xsj/canopy-backend/internal/deliverable/interface/http/v1"
	dischttp "github.com/0xsj/canopy-backend/internal/discussion/interface/http/v1"
	exphttp "github.com/0xsj/canopy-backend/internal/exploration/interface/http/v1"
	identityhttp "github.com/0xsj/canopy-backend/internal/identity/interface/http/v1"
	ledgerhttp "github.com/0xsj/canopy-backend/internal/ledger/interface/http/v1"
	notifhttp "github.com/0xsj/canopy-backend/internal/notification/interface/http/v1"
	orghttp "github.com/0xsj/canopy-backend/internal/organization/interface/http/v1"
	seedhttp "github.com/0xsj/canopy-backend/internal/seed/interface/http/v1"
	sesshttp "github.com/0xsj/canopy-backend/internal/session/interface/http/v1"
	synthhttp "github.com/0xsj/canopy-backend/internal/synthesis/interface/http/v1"
	wshttp "github.com/0xsj/canopy-backend/internal/workspace/interface/http/v1"

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

	// ── 7. Repositories ──────────────────────────────────────────
	dbtx := db.DBTX()

	userRepo := identitypg.NewUserRepository(dbtx)
	orgRepo := orgpg.NewOrgRepository(dbtx)
	orgMemberRepo := orgpg.NewMemberRepository(dbtx)
	teamRepo := orgpg.NewTeamRepository(dbtx)
	wsRepo := wspg.NewWorkspaceRepository(dbtx)
	wsMemberRepo := wspg.NewWorkspaceMemberRepository(dbtx)
	seedRepo := seedpg.NewSeedRepository(dbtx)
	leafRepo := exppg.NewLeafRepository(dbtx)
	branchRepo := exppg.NewBranchRepository(dbtx)
	connRepo := exppg.NewConnectionRepository(dbtx)
	threadRepo := discpg.NewThreadRepository(dbtx)
	signalRepo := convpg.NewSignalRepository(dbtx)
	checkpointRepo := convpg.NewCheckpointRepository(dbtx)
	sessionRepo := sesspg.NewSessionRepository(dbtx)
	synthesisRepo := synthpg.NewSynthesisRepository(dbtx)
	deliverableRepo := delpg.NewDeliverableRepository(dbtx)
	notifRepo := notifpg.NewNotificationRepository(dbtx)
	notifSubRepo := notifpg.NewSubscriptionRepository(dbtx)
	ledgerRepo := ledgerpg.NewLedgerRepository(dbtx)

	// ── 8. Cross-context adapters ────────────────────────────────
	userReader := &userReaderAdapter{repo: userRepo}
	leafWriter := &leafWriterAdapter{repo: leafRepo}
	assembler := noopContextAssembler{}

	// ── 9. Services ──────────────────────────────────────────────
	identitySvc := identitysvc.New(userRepo, pub, log)

	orgSvc := orgsvc.New(
		orgRepo, orgMemberRepo, teamRepo,
		userReader,
		db,
		func(tx database.DBTX) orgdomain.OrgRepository { return orgpg.NewOrgRepository(tx) },
		func(tx database.DBTX) orgdomain.MemberRepository { return orgpg.NewMemberRepository(tx) },
		pub, log,
	)

	wsSvc := wssvc.New(
		wsRepo, wsMemberRepo,
		orgMemberRepo, // satisfies OrgMemberReader (same FindMember shape)
		db,
		func(tx database.DBTX) wsdomain.WorkspaceRepository { return wspg.NewWorkspaceRepository(tx) },
		func(tx database.DBTX) wsdomain.WorkspaceMemberRepository {
			return wspg.NewWorkspaceMemberRepository(tx)
		},
		pub, log,
	)

	seedSvc := seedsvc.New(seedRepo, wsMemberRepo, pub, log)

	expSvc := expsvc.New(
		leafRepo, branchRepo, connRepo,
		wsMemberRepo, // satisfies WorkspaceMemberReader
		db,
		func(tx database.DBTX) expdomain.LeafRepository { return exppg.NewLeafRepository(tx) },
		func(tx database.DBTX) expdomain.BranchRepository { return exppg.NewBranchRepository(tx) },
		pub, log,
	)

	discSvc := discsvc.New(threadRepo, wsMemberRepo, pub, log)

	convSvc := convsvc.New(
		signalRepo, checkpointRepo,
		leafWriter,
		wsMemberRepo,
		pub, log,
	)

	sessSvc := sesssvc.New(sessionRepo, assembler, wsMemberRepo, pub, log)
	synthSvc := synthsvc.New(synthesisRepo, wsMemberRepo, pub, log)
	delSvc := delsvc.New(deliverableRepo, wsMemberRepo, pub, log)
	notifSvc := notifsvc.New(notifRepo, notifSubRepo, nil, pub, log)
	ledgerSvc := ledgersvc.New(ledgerRepo, log)

	// ── 10. HTTP handlers ────────────────────────────────────────
	identityH := identityhttp.NewHandler(identitySvc, log)
	orgH := orghttp.NewHandler(orgSvc, log)
	wsH := wshttp.NewHandler(wsSvc, log)
	seedH := seedhttp.NewHandler(seedSvc, log)
	expH := exphttp.NewHandler(expSvc, log)
	discH := dischttp.NewHandler(discSvc, log)
	convH := convhttp.NewHandler(convSvc, log)
	sessH := sesshttp.NewHandler(sessSvc, log)
	synthH := synthhttp.NewHandler(synthSvc, log)
	delH := delhttp.NewHandler(delSvc, log)
	notifH := notifhttp.NewHandler(notifSvc, log)
	ledgerH := ledgerhttp.NewHandler(ledgerSvc, log)

	// ── 11. Routes ───────────────────────────────────────────────
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
	identityH.Register(apiMux)
	orgH.Register(apiMux)
	wsH.Register(apiMux)
	seedH.Register(apiMux)
	expH.Register(apiMux)
	discH.Register(apiMux)
	convH.Register(apiMux)
	sessH.Register(apiMux)
	synthH.Register(apiMux)
	delH.Register(apiMux)
	notifH.Register(apiMux)
	ledgerH.Register(apiMux)

	mux.Handle("/api/", auth.Middleware(validator, log)(apiMux))

	// ── 12. Middleware ───────────────────────────────────────────
	handler := httpserver.Chain(
		httpserver.Recovery(log),
		httpserver.RequestID(),
		httpserver.CORS(httpserver.CORSConfig{
			AllowedOrigins: []string{"*"},
		}),
		httpserver.Logging(log),
	)(mux)

	// ── 13. Start server + graceful shutdown ─────────────────────
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
