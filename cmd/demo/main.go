package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/0xsj/canopy-backend/pkg/config"
	"github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- Example config sections ---

type ServerConfig struct {
	Host        string
	Port        int
	Environment string
}

func (c *ServerConfig) Load(env config.EnvReader) {
	c.Host = env.String("HOST", "0.0.0.0")
	c.Port = env.Int("PORT", 8080)
	c.Environment = env.String("ENV", "development")
}

func (c *ServerConfig) Validate() error {
	var v config.Errors
	v.Required("host", c.Host)
	v.PortRange("port", c.Port)
	v.OneOf("env", c.Environment, []string{"development", "staging", "production"})
	return v.Err()
}

type DatabaseConfig struct {
	DSN         string
	MaxPoolSize int
}

func (c *DatabaseConfig) Load(env config.EnvReader) {
	c.DSN = env.String("DSN", "postgres://localhost:5432/canopy?sslmode=disable")
	c.MaxPoolSize = env.Int("MAX_POOL_SIZE", 10)
}

func (c *DatabaseConfig) Validate() error {
	var v config.Errors
	v.Required("dsn", c.DSN)
	v.Positive("max_pool_size", c.MaxPoolSize)
	return v.Err()
}

// --- Domain entity ---

type Leaf struct {
	ID          types.LeafID      `json:"id"`
	Title       string            `json:"title"`
	Summary     string            `json:"summary"`
	WorkspaceID types.WorkspaceID `json:"workspace_id"`
	AuthorID    types.UserID      `json:"author_id"`
	Timestamps  types.Timestamps  `json:"timestamps"`
}

// --- Domain layer (repository) ---

func findLeafByID(id types.LeafID) (*Leaf, error) {
	// Simulates a database lookup that finds nothing.
	return nil, errors.New("leaf not found").
		WithKind(errors.KindNotFound).
		WithCode("exploration_leaf_not_found").
		WithSeverity(errors.SeverityLow).
		WithMetadata("leaf_id", id.String())
}

func listLeaves(workspaceID types.WorkspaceID, page types.PageRequest) ([]Leaf, error) {
	// Simulates a database query returning a page of leaves.
	leaves := make([]Leaf, 0, page.EffectiveLimit()+1)
	for i := range page.EffectiveLimit() + 1 {
		leaves = append(leaves, Leaf{
			ID:          types.NewLeafID(),
			Title:       fmt.Sprintf("Leaf #%d", i+1),
			Summary:     "An immutable thought captured during exploration",
			WorkspaceID: workspaceID,
			AuthorID:    types.NewUserID(),
			Timestamps:  types.NewTimestamps(), // immutable — no UpdatedAt
		})
	}
	return leaves, nil
}

// --- Service layer ---

func getLeaf(ctx context.Context, id types.LeafID, wsID types.WorkspaceID) (*Leaf, error) {
	log := logger.FromContext(ctx)
	log.Debug("looking up leaf", logger.String("leaf_id", id.String()))

	leaf, err := findLeafByID(id)
	if err != nil {
		log.Warn("leaf lookup failed",
			logger.String("leaf_id", id.String()),
			logger.Err(err),
		)
		return nil, errors.Wrap(err, "exploration: get leaf").
			WithMetadata("workspace_id", wsID.String())
	}

	return leaf, nil
}

func getLeaves(ctx context.Context, wsID types.WorkspaceID, page types.PageRequest) (*types.PageResponse[Leaf], error) {
	log := logger.FromContext(ctx)
	limit := page.EffectiveLimit()
	log.Debug("listing leaves",
		logger.String("workspace_id", wsID.String()),
		logger.Int("limit", limit),
	)

	// Fetch limit+1 to detect if there are more.
	leaves, err := listLeaves(wsID, page)
	if err != nil {
		return nil, errors.Wrap(err, "exploration: list leaves")
	}

	result := types.NewPageResponse(leaves, limit, func(l Leaf) string {
		return types.EncodeCursor(l.ID.String())
	})

	log.Info("leaves listed",
		logger.Int("count", len(result.Items)),
		logger.Bool("has_more", result.HasMore),
	)
	return &result, nil
}

// --- Handler layer ---

func handleGetLeaf(ctx context.Context) {
	log := logger.FromContext(ctx)
	leafID := types.NewLeafID()
	wsID := types.NewWorkspaceID()

	log.Info("GET /leaf/:id", logger.String("leaf_id", leafID.String()))

	_, err := getLeaf(ctx, leafID, wsID)
	if err != nil {
		meta := errors.CollectMetadata(err)
		origin := errors.OriginFrame(err)
		fields := []logger.Field{
			logger.String("kind", errors.GetKind(err).String()),
			logger.String("code", errors.GetCode(err).String()),
			logger.String("origin", origin.Short()),
		}
		for k, v := range meta {
			fields = append(fields, logger.Any(k, v))
		}
		log.Error("request failed", fields...)

		resp := types.Fail[Leaf](
			errors.GetCode(err).String(),
			"leaf not found",
		)
		printJSON("Error response", resp)
		return
	}
}

func handleCreateLeaf(ctx context.Context) {
	log := logger.FromContext(ctx)

	// Simulate a request with invalid input.
	title := ""
	summary := ""

	log.Info("POST /leaves", logger.String("title", title))

	// Validate input.
	var fieldErrors []types.FieldError
	if title == "" {
		fieldErrors = append(fieldErrors, types.FieldError{
			Field:   "title",
			Message: "is required",
		})
	}
	if summary == "" {
		fieldErrors = append(fieldErrors, types.FieldError{
			Field:   "summary",
			Message: "is required",
		})
	}

	if len(fieldErrors) > 0 {
		log.Warn("validation failed", logger.Int("field_errors", len(fieldErrors)))

		resp := types.FailWithDetails[Leaf](
			"validation_failed",
			"one or more fields are invalid",
			fieldErrors,
		)
		printJSON("Validation error response", resp)
		return
	}
}

func handleListLeaves(ctx context.Context) {
	log := logger.FromContext(ctx)
	wsID := types.NewWorkspaceID()
	page := types.PageRequest{Limit: 3}

	log.Info("GET /leaves", logger.String("workspace_id", wsID.String()))

	result, err := getLeaves(ctx, wsID, page)
	if err != nil {
		log.Error("request failed", logger.Err(err))
		return
	}

	resp := types.OK(*result)
	printJSON("Paginated response", resp)
}

func printJSON(label string, v any) {
	fmt.Println()
	fmt.Printf("=== %s ===\n", label)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func main() {
	// --- 1. Config ---
	var serverCfg ServerConfig
	var dbCfg DatabaseConfig

	loader := config.NewLoader("CANOPY")
	loader.Register("SERVER", &serverCfg)
	loader.Register("DATABASE", &dbCfg)

	if err := loader.LoadAll(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}

	// --- 2. Logger ---
	log := logger.NewConsole(
		logger.WithLevel(logger.LevelDebug),
		logger.WithColor(true),
		logger.WithTimestamps(true),
	)

	log.Info("config loaded",
		logger.String("host", serverCfg.Host),
		logger.Int("port", serverCfg.Port),
		logger.String("env", serverCfg.Environment),
	)

	addr := fmt.Sprintf("%s:%d", serverCfg.Host, serverCfg.Port)
	log.Info("server started", logger.String("addr", addr))

	// --- 3. Request simulations ---
	requestLog := log.With(logger.String("request_id", "req-8f3a"))
	ctx := logger.WithContext(context.Background(), requestLog)

	fmt.Println()
	log.Info("--- simulate: GET /leaf/:id (not found) ---")
	handleGetLeaf(ctx)

	fmt.Println()
	log.Info("--- simulate: POST /leaves (validation error) ---")
	handleCreateLeaf(ctx)

	fmt.Println()
	log.Info("--- simulate: GET /leaves (paginated) ---")
	handleListLeaves(ctx)
}
