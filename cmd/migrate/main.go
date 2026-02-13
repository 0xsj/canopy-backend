package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/0xsj/canopy-backend/pkg/config"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"

	convergencepg "github.com/0xsj/canopy-backend/internal/convergence/adapter/postgres"
	deliverablepg "github.com/0xsj/canopy-backend/internal/deliverable/adapter/postgres"
	discussionpg "github.com/0xsj/canopy-backend/internal/discussion/adapter/postgres"
	explorationpg "github.com/0xsj/canopy-backend/internal/exploration/adapter/postgres"
	identitypg "github.com/0xsj/canopy-backend/internal/identity/adapter/postgres"
	ledgerpg "github.com/0xsj/canopy-backend/internal/ledger/adapter/postgres"
	notificationpg "github.com/0xsj/canopy-backend/internal/notification/adapter/postgres"
	organizationpg "github.com/0xsj/canopy-backend/internal/organization/adapter/postgres"
	seedpg "github.com/0xsj/canopy-backend/internal/seed/adapter/postgres"
	sessionpg "github.com/0xsj/canopy-backend/internal/session/adapter/postgres"
	synthesispg "github.com/0xsj/canopy-backend/internal/synthesis/adapter/postgres"
	workspacepg "github.com/0xsj/canopy-backend/internal/workspace/adapter/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── 1. Config ────────────────────────────────────────────────
	var dbCfg database.Config

	loader := config.NewLoader("CANOPY")
	loader.Register("DATABASE", &dbCfg)

	if err := loader.LoadAll(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}

	// ── 2. Logger ────────────────────────────────────────────────
	log := logger.NewConsole(
		logger.WithLevel(logger.LevelInfo),
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

	// ── 4. Migration set ─────────────────────────────────────────
	set := database.NewMigrationSet()
	set.Add("identity", identitypg.MigrationFS())
	set.Add("organization", organizationpg.MigrationFS())
	set.Add("workspace", workspacepg.MigrationFS())
	set.Add("seed", seedpg.MigrationFS())
	set.Add("exploration", explorationpg.MigrationFS())
	set.Add("discussion", discussionpg.MigrationFS())
	set.Add("convergence", convergencepg.MigrationFS())
	set.Add("session", sessionpg.MigrationFS())
	set.Add("synthesis", synthesispg.MigrationFS())
	set.Add("deliverable", deliverablepg.MigrationFS())
	set.Add("notification", notificationpg.MigrationFS())
	set.Add("ledger", ledgerpg.MigrationFS())

	// ── 5. Run migrations ────────────────────────────────────────
	log.Info("running migrations for 12 schemas")

	if err := set.RunAll(ctx, db, log); err != nil {
		log.Error("migration failed", logger.Err(err))
		os.Exit(1)
	}

	log.Info("all migrations applied successfully")
}
