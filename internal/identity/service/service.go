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

// Service implements the identity application logic.
type Service struct {
	repo domain.UserRepository
	pub  events.Publisher
	log  logger.Logger
}

// New creates a new identity service.
func New(repo domain.UserRepository, pub events.Publisher, log logger.Logger) *Service {
	return &Service{repo: repo, pub: pub, log: log}
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
