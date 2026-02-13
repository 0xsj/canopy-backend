-- name: CreateNotification :exec
INSERT INTO notifications (id, user_id, channel, title, body, resource_type, resource_id, workspace_id, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: FindNotificationsByUser :many
SELECT id, user_id, channel, title, body, resource_type, resource_id, workspace_id, status, created_at, updated_at
FROM notifications WHERE user_id = $1
ORDER BY created_at DESC LIMIT $2;

-- name: FindUnreadNotificationsByUser :many
SELECT id, user_id, channel, title, body, resource_type, resource_id, workspace_id, status, created_at, updated_at
FROM notifications WHERE user_id = $1 AND status != 'read'
ORDER BY created_at DESC;

-- name: UpdateNotification :execresult
UPDATE notifications SET status = $2, updated_at = $3 WHERE id = $1;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET status = 'read', updated_at = NOW()
WHERE user_id = $1 AND status != 'read';

-- name: SaveSubscription :exec
INSERT INTO subscriptions (user_id, workspace_id, channels, digest_frequency, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, workspace_id) DO UPDATE
SET channels = $3, digest_frequency = $4, updated_at = $6;

-- name: FindSubscriptionByUserAndWorkspace :one
SELECT user_id, workspace_id, channels, digest_frequency, created_at, updated_at
FROM subscriptions WHERE user_id = $1 AND workspace_id = $2;

-- name: FindSubscriptionsByWorkspace :many
SELECT user_id, workspace_id, channels, digest_frequency, created_at, updated_at
FROM subscriptions WHERE workspace_id = $1;
