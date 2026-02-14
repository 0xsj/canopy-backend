package service

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/identity/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// Encryptor provides symmetric encryption for sensitive data (API keys).
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// Service implements the identity application logic.
type Service struct {
	repo       domain.UserRepository
	llmConfigs domain.UserLLMConfigRepository
	encryptor  Encryptor
	pub        events.Publisher
	log        logger.Logger
}

// New creates a new identity service.
func New(
	repo domain.UserRepository,
	llmConfigs domain.UserLLMConfigRepository,
	encryptor Encryptor,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	return &Service{
		repo:       repo,
		llmConfigs: llmConfigs,
		encryptor:  encryptor,
		pub:        pub,
		log:        log,
	}
}

// Register creates a user record on first login. If the user already exists
// (by external ID), the existing record is returned.
func (s *Service) Register(ctx context.Context, externalID, email, displayName string) (domain.User, error) {
	const op = "identity: register"

	existing, err := s.repo.FindByExternalID(ctx, externalID)
	if err == nil {
		return existing, nil
	}
	if canopyerr.GetKind(err) != canopyerr.KindNotFound {
		return domain.User{}, canopyerr.Wrap(err, op)
	}

	user, err := domain.NewUser(externalID, displayName, email)
	if err != nil {
		return domain.User{}, canopyerr.Wrap(err, op)
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return domain.User{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectUserRegistered, "", domain.UserRegisteredData{
		UserID:      user.ID().String(),
		ExternalID:  user.ExternalID(),
		DisplayName: user.DisplayName(),
		Email:       user.Email(),
		Timestamp:   time.Now().UTC(),
	})

	s.log.Info("user registered",
		logger.String("user_id", user.ID().String()),
		logger.String("email", user.Email()),
	)

	return user, nil
}

// FindByID returns a user by internal ID.
func (s *Service) FindByID(ctx context.Context, id types.UserID) (domain.User, error) {
	const op = "identity: find by id"
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return domain.User{}, canopyerr.Wrap(err, op)
	}
	return user, nil
}

// FindByExternalID returns a user by OAuth provider subject identifier.
func (s *Service) FindByExternalID(ctx context.Context, externalID string) (domain.User, error) {
	const op = "identity: find by external id"
	user, err := s.repo.FindByExternalID(ctx, externalID)
	if err != nil {
		return domain.User{}, canopyerr.Wrap(err, op)
	}
	return user, nil
}

// UpdateProfile updates the authenticated user's profile.
func (s *Service) UpdateProfile(ctx context.Context, displayName, email, avatarURL string) (domain.User, error) {
	const op = "identity: update profile"

	claims, ok := auth.FromClaims(ctx)
	if !ok {
		return domain.User{}, canopyerr.Wrap(canopyerr.ErrUnauthenticated, op)
	}

	user, err := s.repo.FindByExternalID(ctx, claims.Subject)
	if err != nil {
		return domain.User{}, canopyerr.Wrap(err, op)
	}

	if err := user.UpdateProfile(displayName, email, avatarURL); err != nil {
		return domain.User{}, canopyerr.Wrap(err, op)
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return domain.User{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectUserProfileUpdated, "", domain.UserProfileUpdatedData{
		UserID:      user.ID().String(),
		DisplayName: user.DisplayName(),
		Email:       user.Email(),
		AvatarURL:   user.AvatarURL(),
		Timestamp:   time.Now().UTC(),
	})

	return user, nil
}

// --- User LLM Config ---

// UserLLMConfigResult holds a decrypted user LLM config for the handler to render.
type UserLLMConfigResult struct {
	UserID     types.UserID
	Provider   string
	Model      string
	APIKey     string // decrypted plaintext, masked by handler
	Timestamps types.Timestamps
}

// SetLLMConfig sets or updates the authenticated user's personal LLM configuration.
func (s *Service) SetLLMConfig(ctx context.Context, provider, model, apiKey string) error {
	const op = "identity: set llm config"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	enc, err := s.encryptor.Encrypt([]byte(apiKey))
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	cfg, err := domain.NewUserLLMConfig(callerID, provider, model, enc)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.llmConfigs.Upsert(ctx, cfg); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectUserLLMConfigUpdated, "", domain.UserLLMConfigUpdatedData{
		UserID:    callerID.String(),
		Provider:  provider,
		Model:     model,
		Timestamp: time.Now().UTC(),
	})

	s.log.Info("user llm config updated",
		logger.String("user_id", callerID.String()),
		logger.String("provider", provider),
		logger.String("model", model),
	)

	return nil
}

// GetLLMConfig returns the authenticated user's personal LLM configuration.
func (s *Service) GetLLMConfig(ctx context.Context) (UserLLMConfigResult, error) {
	const op = "identity: get llm config"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return UserLLMConfigResult{}, canopyerr.Wrap(err, op)
	}

	cfg, err := s.llmConfigs.FindByUser(ctx, callerID)
	if err != nil {
		return UserLLMConfigResult{}, canopyerr.Wrap(err, op)
	}

	plainKey, err := s.encryptor.Decrypt(cfg.APIKeyEnc())
	if err != nil {
		return UserLLMConfigResult{}, canopyerr.Wrap(err, op)
	}

	return UserLLMConfigResult{
		UserID:     cfg.UserID(),
		Provider:   string(cfg.Provider()),
		Model:      cfg.Model(),
		APIKey:     string(plainKey),
		Timestamps: cfg.Timestamps(),
	}, nil
}

// DeleteLLMConfig removes the authenticated user's personal LLM configuration.
func (s *Service) DeleteLLMConfig(ctx context.Context) error {
	const op = "identity: delete llm config"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.llmConfigs.Delete(ctx, callerID); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectUserLLMConfigDeleted, "", domain.UserLLMConfigDeletedData{
		UserID:    callerID.String(),
		Timestamp: time.Now().UTC(),
	})

	s.log.Info("user llm config deleted",
		logger.String("user_id", callerID.String()),
	)

	return nil
}

// --- Auth Helpers ---

func (s *Service) authenticatedUserID(ctx context.Context) (types.UserID, error) {
	claims, ok := auth.FromClaims(ctx)
	if !ok {
		return types.UserID{}, canopyerr.ErrUnauthenticated
	}
	return types.UserIDFrom(claims.Subject), nil
}

// publish is a fire-and-forget helper. Failures are logged, not returned.
func (s *Service) publish(ctx context.Context, eventType, workspaceID string, data any) {
	event, err := events.New(eventType, workspaceID, data)
	if err != nil {
		s.log.Error("event creation failed", logger.String("type", eventType), logger.Err(err))
		return
	}
	event.Subject = events.BuildSubject(workspaceID, "identity", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
