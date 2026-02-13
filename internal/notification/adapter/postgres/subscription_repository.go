package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/notification/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// SubscriptionRepository implements domain.SubscriptionRepository using Postgres.
type SubscriptionRepository struct {
	db database.DBTX
}

// NewSubscriptionRepository creates a new SubscriptionRepository.
func NewSubscriptionRepository(db database.DBTX) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

var _ domain.SubscriptionRepository = (*SubscriptionRepository)(nil)

func (r *SubscriptionRepository) Save(ctx context.Context, sub domain.Subscription) error {
	const op = "notification: save subscription"
	const query = `
		INSERT INTO subscriptions (user_id, workspace_id, channels, digest_frequency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, workspace_id) DO UPDATE
		SET channels = $3, digest_frequency = $4, updated_at = $6`

	ts := sub.Timestamps()
	channels := make([]string, len(sub.Channels()))
	for i, ch := range sub.Channels() {
		channels[i] = string(ch)
	}

	_, err := r.db.Exec(ctx, query,
		sub.UserID().String(),
		sub.WorkspaceID().String(),
		channels,
		string(sub.DigestFrequency()),
		ts.CreatedAt.Time(),
		tsUpdatedAtSub(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *SubscriptionRepository) FindByUserAndWorkspace(ctx context.Context, userID types.UserID, workspaceID types.WorkspaceID) (domain.Subscription, error) {
	const op = "notification: find subscription by user and workspace"
	const query = `
		SELECT user_id, workspace_id, channels, digest_frequency, created_at, updated_at
		FROM subscriptions WHERE user_id = $1 AND workspace_id = $2`

	return scanSubscription(r.db.QueryRow(ctx, query, userID.String(), workspaceID.String()), op)
}

func (r *SubscriptionRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Subscription, error) {
	const op = "notification: find subscriptions by workspace"
	const query = `
		SELECT user_id, workspace_id, channels, digest_frequency, created_at, updated_at
		FROM subscriptions WHERE workspace_id = $1`

	rows, err := r.db.Query(ctx, query, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var subs []domain.Subscription
	for rows.Next() {
		s, err := scanSubscription(rows, op)
		if err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// --- helpers ---

func scanSubscription(row rowScanner, op string) (domain.Subscription, error) {
	var (
		rawUserID       string
		rawWorkspaceID  string
		channelStrs     []string
		digestFrequency string
		createdAt       time.Time
		updatedAt       time.Time
	)

	if err := row.Scan(&rawUserID, &rawWorkspaceID, &channelStrs, &digestFrequency, &createdAt, &updatedAt); err != nil {
		return domain.Subscription{}, database.MapQueryError(err, op)
	}

	channels := make([]domain.Channel, len(channelStrs))
	for i, s := range channelStrs {
		channels[i] = domain.Channel(s)
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructSubscription(
		types.UserIDFrom(rawUserID),
		types.WorkspaceIDFrom(rawWorkspaceID),
		channels,
		domain.DigestFrequency(digestFrequency),
		types.Timestamps{
			CreatedAt: types.TimestampFrom(createdAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func tsUpdatedAtSub(ts types.Timestamps) time.Time {
	if ts.UpdatedAt != nil {
		return ts.UpdatedAt.Time()
	}
	return ts.CreatedAt.Time()
}
