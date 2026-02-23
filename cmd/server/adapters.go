package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"

	delsvc "github.com/0xsj/canopy-backend/internal/deliverable/service"
	expdomain "github.com/0xsj/canopy-backend/internal/exploration/domain"
	identitydomain "github.com/0xsj/canopy-backend/internal/identity/domain"
	orgsvc "github.com/0xsj/canopy-backend/internal/organization/service"
	seeddomain "github.com/0xsj/canopy-backend/internal/seed/domain"
	synthsvc "github.com/0xsj/canopy-backend/internal/synthesis/service"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	"github.com/0xsj/canopy-backend/pkg/crypto"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/llm"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
	"github.com/0xsj/canopy-backend/pkg/websocket"
)

// streamBroadcasterAdapter implements llm.StreamBroadcaster by forwarding
// stream messages to the WebSocket hub.
type streamBroadcasterAdapter struct {
	hub *websocket.Hub
}

func (a *streamBroadcasterAdapter) Send(workspaceID string, msg llm.StreamMessage) error {
	wsMsg, err := websocket.NewMessage(msg.Type, workspaceID, msg)
	if err != nil {
		return err
	}
	return a.hub.BroadcastMessage(workspaceID, wsMsg)
}

// userReaderAdapter wraps the identity UserRepository to satisfy the
// organization service's UserReader cross-context port.
type userReaderAdapter struct {
	repo identitydomain.UserRepository
}

func (a *userReaderAdapter) FindByID(ctx context.Context, id types.UserID) (orgsvc.UserInfo, error) {
	user, err := a.repo.FindByID(ctx, id)
	if err != nil {
		return orgsvc.UserInfo{}, err
	}
	return orgsvc.UserInfo{
		ID:          user.ID(),
		DisplayName: user.DisplayName(),
		Email:       user.Email(),
	}, nil
}

// leafWriterAdapter wraps the exploration LeafRepository to satisfy the
// convergence service's LeafWriter cross-context port (string → Layer).
type leafWriterAdapter struct {
	repo expdomain.LeafRepository
}

func (a *leafWriterAdapter) UpdateLayer(ctx context.Context, id types.LeafID, layer string) error {
	return a.repo.UpdateLayer(ctx, id, expdomain.Layer(layer))
}

// sourceLeafReaderAdapter wraps the exploration LeafRepository to satisfy the
// synthesis service's SourceLeafReader cross-context port.
type sourceLeafReaderAdapter struct {
	repo expdomain.LeafRepository
}

func (a *sourceLeafReaderAdapter) FindByIDs(ctx context.Context, ids []types.LeafID) ([]synthsvc.SourceLeaf, error) {
	leaves, err := a.repo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]synthsvc.SourceLeaf, len(leaves))
	for i, l := range leaves {
		out[i] = synthsvc.SourceLeaf{
			ID:            l.ID(),
			SeedID:        l.SeedID(),
			Title:         l.Title(),
			Summary:       l.Summary(),
			KeyPoints:     l.KeyPoints(),
			OpenQuestions: l.OpenQuestions(),
			Tags:          l.Tags(),
		}
	}
	return out, nil
}

// seedReaderForSynthesisAdapter wraps the seed SeedRepository to satisfy the
// synthesis service's SeedReader cross-context port.
type seedReaderForSynthesisAdapter struct {
	repo seeddomain.SeedRepository
}

func (a *seedReaderForSynthesisAdapter) FindByID(ctx context.Context, id types.SeedID) (synthsvc.SeedInfo, error) {
	seed, err := a.repo.FindByID(ctx, id)
	if err != nil {
		return synthsvc.SeedInfo{}, err
	}
	return synthsvc.SeedInfo{
		Title:       seed.Title(),
		Description: seed.Description(),
	}, nil
}

// synthesisLeafCreatorAdapter creates a synthesis leaf in the exploration
// context. It creates a branch and leaf atomically in a transaction.
type synthesisLeafCreatorAdapter struct {
	db            *database.DB
	newLeafRepo   func(database.DBTX) expdomain.LeafRepository
	newBranchRepo func(database.DBTX) expdomain.BranchRepository
}

func (a *synthesisLeafCreatorAdapter) Create(ctx context.Context, params synthsvc.SynthesisLeafParams) (types.LeafID, error) {
	branch, err := expdomain.NewBranch(params.WorkspaceID, params.SeedID, params.AuthorID)
	if err != nil {
		return types.LeafID{}, err
	}

	sources := make([]expdomain.Source, len(params.Sources))
	for i, s := range params.Sources {
		sources[i] = expdomain.Source{LeafID: s.LeafID, Title: s.Title}
	}

	leaf, err := expdomain.NewSynthesisLeaf(
		params.WorkspaceID, params.SeedID, branch.ID(), params.AuthorID,
		params.Title, params.Summary, params.KeyPoints, params.OpenQuestions, params.Tags,
		sources,
	)
	if err != nil {
		return types.LeafID{}, err
	}

	if err := a.db.WithTx(ctx, func(tx pgx.Tx) error {
		txBranches := a.newBranchRepo(tx)
		txLeaves := a.newLeafRepo(tx)

		if err := txBranches.Create(ctx, branch); err != nil {
			return err
		}
		if err := txLeaves.Create(ctx, leaf); err != nil {
			return err
		}
		if err := branch.SetRootLeaf(leaf.ID()); err != nil {
			return err
		}
		return txBranches.Update(ctx, branch)
	}); err != nil {
		return types.LeafID{}, err
	}

	return leaf.ID(), nil
}

// deliverableLeafReaderAdapter wraps the exploration LeafRepository to satisfy
// the deliverable service's SourceLeafReader cross-context port.
type deliverableLeafReaderAdapter struct {
	repo expdomain.LeafRepository
}

func (a *deliverableLeafReaderAdapter) FindByIDs(ctx context.Context, ids []types.LeafID) ([]delsvc.SourceLeaf, error) {
	leaves, err := a.repo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]delsvc.SourceLeaf, len(leaves))
	for i, l := range leaves {
		out[i] = delsvc.SourceLeaf{
			ID:            l.ID(),
			Title:         l.Title(),
			Summary:       l.Summary(),
			KeyPoints:     l.KeyPoints(),
			OpenQuestions: l.OpenQuestions(),
			Tags:          l.Tags(),
		}
	}
	return out, nil
}

// workspaceConfigReaderAdapter wraps the workspace WorkspaceRepository to satisfy
// the deliverable service's WorkspaceConfigReader cross-context port.
type workspaceConfigReaderAdapter struct {
	repo wsdomain.WorkspaceRepository
}

func (a *workspaceConfigReaderAdapter) Configuration(ctx context.Context, workspaceID types.WorkspaceID) (map[string]any, error) {
	ws, err := a.repo.FindByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return ws.Configuration(), nil
}

// ledgerMemberReaderAdapter wraps the workspace MemberRepository to satisfy
// the ledger service's WorkspaceMemberReader cross-context port.
type ledgerMemberReaderAdapter struct {
	repo wsdomain.WorkspaceMemberRepository
}

func (a *ledgerMemberReaderAdapter) FindMember(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error) {
	return a.repo.FindMember(ctx, workspaceID, userID)
}

// userLLMConfigReader is a cross-context read port for per-user LLM configs.
type userLLMConfigReader interface {
	FindByUser(ctx context.Context, userID types.UserID) (identitydomain.UserLLMConfig, error)
}

// llmProviderResolver resolves an LLM provider with hierarchy: User > Workspace > Server.
// If the authenticated user has a personal LLM config, that is used. Otherwise, the
// workspace config is tried. If neither exists, the server-wide fallback is returned.
type llmProviderResolver struct {
	wsConfigs   wsdomain.LLMConfigRepository
	userConfigs userLLMConfigReader
	cipher      crypto.Encryptor
	fallback    llm.Provider
	log         logger.Logger

	mu    sync.RWMutex
	cache map[string]llm.Provider // keyed by "u:<userID>:w:<wsID>" or "w:<wsID>"
}

func newLLMProviderResolver(
	wsConfigs wsdomain.LLMConfigRepository,
	userConfigs userLLMConfigReader,
	cipher crypto.Encryptor,
	fallback llm.Provider,
	log logger.Logger,
) *llmProviderResolver {
	return &llmProviderResolver{
		wsConfigs:   wsConfigs,
		userConfigs: userConfigs,
		cipher:      cipher,
		fallback:    fallback,
		log:         log,
		cache:       make(map[string]llm.Provider),
	}
}

func (r *llmProviderResolver) Resolve(ctx context.Context, workspaceID types.WorkspaceID) (llm.Provider, error) {
	wsKey := workspaceID.String()

	// Try user-level config first.
	claims, hasAuth := auth.FromClaims(ctx)
	if hasAuth {
		userID := types.UserIDFrom(claims.Subject)
		cacheKey := "u:" + userID.String() + ":w:" + wsKey

		r.mu.RLock()
		if p, ok := r.cache[cacheKey]; ok {
			r.mu.RUnlock()
			return p, nil
		}
		r.mu.RUnlock()

		userCfg, err := r.userConfigs.FindByUser(ctx, userID)
		if err == nil {
			provider, cErr := r.createAndCache(ctx, cacheKey,
				string(userCfg.Provider()), userCfg.Model(), userCfg.APIKeyEnc())
			if cErr != nil {
				return nil, fmt.Errorf("resolve llm provider (user): %w", cErr)
			}
			r.log.Debug("user llm provider cached",
				logger.String("user_id", userID.String()),
				logger.String("provider", string(userCfg.Provider())),
			)
			return provider, nil
		}
		if canopyerr.GetKind(err) != canopyerr.KindNotFound {
			return nil, fmt.Errorf("resolve llm provider: user config: %w", err)
		}
		// User has no config — fall through to workspace.
	}

	// Try workspace-level config.
	wsCacheKey := "w:" + wsKey

	r.mu.RLock()
	if p, ok := r.cache[wsCacheKey]; ok {
		r.mu.RUnlock()
		return p, nil
	}
	r.mu.RUnlock()

	wsCfg, err := r.wsConfigs.FindByWorkspace(ctx, workspaceID)
	if err != nil {
		if canopyerr.GetKind(err) == canopyerr.KindNotFound {
			return r.fallback, nil
		}
		return nil, fmt.Errorf("resolve llm provider: %w", err)
	}

	provider, err := r.createAndCache(ctx, wsCacheKey,
		string(wsCfg.Provider()), wsCfg.Model(), wsCfg.APIKeyEnc())
	if err != nil {
		return nil, fmt.Errorf("resolve llm provider (workspace): %w", err)
	}

	r.log.Debug("workspace llm provider cached",
		logger.String("workspace_id", wsKey),
		logger.String("provider", string(wsCfg.Provider())),
	)

	return provider, nil
}

// createAndCache decrypts an API key, creates a provider, and caches it.
func (r *llmProviderResolver) createAndCache(ctx context.Context, cacheKey, provider, model string, apiKeyEnc []byte) (llm.Provider, error) {
	plainKey, err := r.cipher.Decrypt(apiKeyEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt key: %w", err)
	}

	p, err := llm.NewProvider(ctx, llm.Config{
		Provider: provider,
		Model:    model,
		APIKey:   string(plainKey),
	}, r.log)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}

	r.mu.Lock()
	r.cache[cacheKey] = p
	r.mu.Unlock()

	return p, nil
}

// InvalidateCache removes cached providers for a workspace.
// Called when workspace LLM config is updated or deleted.
func (r *llmProviderResolver) InvalidateCache(workspaceID string) {
	r.mu.Lock()
	delete(r.cache, "w:"+workspaceID)
	r.mu.Unlock()
}

// InvalidateUserCache removes all cached providers for a user.
// Called when user LLM config is updated or deleted.
func (r *llmProviderResolver) InvalidateUserCache(userID string) {
	prefix := "u:" + userID + ":"
	r.mu.Lock()
	for key := range r.cache {
		if strings.HasPrefix(key, prefix) {
			delete(r.cache, key)
		}
	}
	r.mu.Unlock()
}
