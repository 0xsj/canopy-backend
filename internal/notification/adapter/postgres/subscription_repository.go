package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/notification/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/notification/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// SubscriptionRepository implements domain.SubscriptionRepository using Postgres via sqlc.
type SubscriptionRepository struct {
	q *sqlc.Queries
}

// NewSubscriptionRepository creates a new SubscriptionRepository.
func NewSubscriptionRepository(db database.DBTX) *SubscriptionRepository {
	return &SubscriptionRepository{q: sqlc.New(db)}
}

var _ domain.SubscriptionRepository = (*SubscriptionRepository)(nil)

func (r *SubscriptionRepository) Save(ctx context.Context, sub domain.Subscription) error {
	const op = "notification: save subscription"
	return database.MapQueryError(r.q.SaveSubscription(ctx, subscriptionToSaveParams(sub)), op)
}

func (r *SubscriptionRepository) FindByUserAndWorkspace(ctx context.Context, userID types.UserID, workspaceID types.WorkspaceID) (domain.Subscription, error) {
	const op = "notification: find subscription by user and workspace"
	row, err := r.q.FindSubscriptionByUserAndWorkspace(ctx, sqlc.FindSubscriptionByUserAndWorkspaceParams{
		UserID:      userID.String(),
		WorkspaceID: workspaceID.String(),
	})
	if err != nil {
		return domain.Subscription{}, database.MapQueryError(err, op)
	}
	return subscriptionToDomain(row), nil
}

func (r *SubscriptionRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Subscription, error) {
	const op = "notification: find subscriptions by workspace"
	rows, err := r.q.FindSubscriptionsByWorkspace(ctx, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return subscriptionsToDomain(rows), nil
}
