package domain

import "time"

// Event subjects published by the notification context.
const (
	SubjectNotificationSent = "notification.sent"
	SubjectDigestGenerated  = "notification.digest.generated"
)

// NotificationSentData is published when a notification is delivered.
type NotificationSentData struct {
	NotificationID string    `json:"notification_id"`
	UserID         string    `json:"user_id"`
	Channel        string    `json:"channel"`
	Title          string    `json:"title"`
	Timestamp      time.Time `json:"timestamp"`
}

// DigestGeneratedData is published when an AI-generated digest is produced.
type DigestGeneratedData struct {
	UserID      string    `json:"user_id"`
	WorkspaceID string    `json:"workspace_id"`
	Timestamp   time.Time `json:"timestamp"`
}
