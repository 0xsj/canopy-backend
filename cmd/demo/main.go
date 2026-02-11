package main

import (
	"context"
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// --- Domain layer (repository) ---

func findLeafByID(id string) error {
	return errors.New("leaf not found").
		WithKind(errors.KindNotFound).
		WithCode("exploration_leaf_not_found").
		WithSeverity(errors.SeverityLow).
		WithMetadata("leaf_id", id)
}

// --- Service layer ---

func getLeaf(ctx context.Context, id string, workspaceID string) error {
	log := logger.FromContext(ctx)
	log.Debug("looking up leaf", logger.String("leaf_id", id))

	err := findLeafByID(id)
	if err != nil {
		log.Warn("leaf lookup failed",
			logger.String("leaf_id", id),
			logger.Err(err),
		)
		return errors.Wrap(err, "exploration: get leaf").
			WithMetadata("workspace_id", workspaceID)
	}

	log.Info("leaf retrieved", logger.String("leaf_id", id))
	return nil
}

// --- Handler layer ---

func handleGetLeaf(ctx context.Context) {
	log := logger.FromContext(ctx)
	leafID := "leaf-abc-123"
	workspaceID := "ws-xyz-789"

	log.Info("handling GET /leaf", logger.String("leaf_id", leafID))

	err := getLeaf(ctx, leafID, workspaceID)
	if err == nil {
		log.Info("success")
		return
	}

	// Log the error with all collected metadata and origin.
	meta := errors.CollectMetadata(err)
	origin := errors.OriginFrame(err)
	fields := []logger.Field{
		logger.String("kind", errors.GetKind(err).String()),
		logger.String("code", errors.GetCode(err).String()),
		logger.String("severity", errors.GetSeverity(err).String()),
		logger.String("origin", origin.Short()),
	}
	for k, v := range meta {
		fields = append(fields, logger.Any(k, v))
	}
	log.Error("request failed", fields...)

	// What the HTTP response would look like.
	fmt.Println()
	fmt.Println("=== HTTP response ===")
	fmt.Printf("Status: 404\n")
	fmt.Printf("Body:   {\"error\": {\"code\": \"%s\", \"message\": \"leaf not found\"}}\n",
		errors.GetCode(err))
}

func main() {
	// Wire the logger — this is what the composition root would do.
	log := logger.NewConsole(
		logger.WithLevel(logger.LevelDebug),
		logger.WithColor(true),
		logger.WithTimestamps(true),
	)

	// Simulate middleware attaching a request-scoped logger to context.
	requestLog := log.With(logger.String("request_id", "req-8f3a"))
	ctx := logger.WithContext(context.Background(), requestLog)

	log.Info("server started", logger.String("addr", ":8080"))
	fmt.Println()

	handleGetLeaf(ctx)
}
