package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Channel represents a notification delivery channel.
type Channel string

const (
	ChannelInApp Channel = "in_app"
	ChannelPush  Channel = "push"
	ChannelEmail Channel = "email"
)

// IsValid reports whether the channel is a recognized value.
func (c Channel) IsValid() bool {
	switch c {
	case ChannelInApp, ChannelPush, ChannelEmail:
		return true
	}
	return false
}

// NotificationStatus tracks delivery state.
type NotificationStatus string

const (
	NotificationPending   NotificationStatus = "pending"
	NotificationDelivered NotificationStatus = "delivered"
	NotificationRead      NotificationStatus = "read"
)

// Notification represents a single notification to a user.
type Notification struct {
	id           types.ID[notificationTag]
	userID       types.UserID
	channel      Channel
	title        string
	body         string
	resourceType string // what the notification is about (e.g., "leaf", "checkpoint")
	resourceID   string
	workspaceID  string // optional context
	status       NotificationStatus
	timestamps   types.Timestamps
}

type notificationTag struct{}

// NotificationID is the exported type alias for use in adapter packages.
type NotificationID = types.ID[notificationTag]

const prefixNotification = "ntf"

// NewNotification creates a new pending notification.
func NewNotification(
	userID types.UserID,
	channel Channel,
	title string,
	body string,
	resourceType string,
	resourceID string,
	workspaceID string,
) (Notification, error) {
	if userID.IsZero() {
		return Notification{}, fmt.Errorf("notification: user ID is required")
	}
	if !channel.IsValid() {
		return Notification{}, fmt.Errorf("notification: invalid channel %q", channel)
	}
	if title == "" {
		return Notification{}, fmt.Errorf("notification: title is required")
	}

	return Notification{
		id:           types.NewID[notificationTag](prefixNotification),
		userID:       userID,
		channel:      channel,
		title:        title,
		body:         body,
		resourceType: resourceType,
		resourceID:   resourceID,
		workspaceID:  workspaceID,
		status:       NotificationPending,
		timestamps:   types.NewMutableTimestamps(),
	}, nil
}

// ReconstructNotification builds a Notification from trusted data.
func ReconstructNotification(
	id types.ID[notificationTag],
	userID types.UserID,
	channel Channel,
	title string,
	body string,
	resourceType string,
	resourceID string,
	workspaceID string,
	status NotificationStatus,
	timestamps types.Timestamps,
) Notification {
	return Notification{
		id:           id,
		userID:       userID,
		channel:      channel,
		title:        title,
		body:         body,
		resourceType: resourceType,
		resourceID:   resourceID,
		workspaceID:  workspaceID,
		status:       status,
		timestamps:   timestamps,
	}
}

// MarkDelivered transitions the notification to delivered status.
func (n *Notification) MarkDelivered() error {
	if n.status != NotificationPending {
		return fmt.Errorf("notification: can only deliver a pending notification, current: %s", n.status)
	}
	n.status = NotificationDelivered
	n.timestamps.Touch()
	return nil
}

// MarkRead transitions the notification to read status.
func (n *Notification) MarkRead() error {
	if n.status == NotificationRead {
		return fmt.Errorf("notification: already read")
	}
	n.status = NotificationRead
	n.timestamps.Touch()
	return nil
}

func (n Notification) ID() types.ID[notificationTag] { return n.id }
func (n Notification) UserID() types.UserID          { return n.userID }
func (n Notification) Channel() Channel              { return n.channel }
func (n Notification) Title() string                 { return n.title }
func (n Notification) Body() string                  { return n.body }
func (n Notification) ResourceType() string          { return n.resourceType }
func (n Notification) ResourceID() string            { return n.resourceID }
func (n Notification) WorkspaceID() string           { return n.workspaceID }
func (n Notification) Status() NotificationStatus    { return n.status }
func (n Notification) Timestamps() types.Timestamps  { return n.timestamps }

// NotificationIDFrom creates a NotificationID from a trusted database string.
func NotificationIDFrom(raw string) types.ID[notificationTag] {
	return types.IDFrom[notificationTag](raw)
}

// ParseNotificationID validates and parses an untrusted notification ID string.
func ParseNotificationID(raw string) (NotificationID, error) {
	return types.ParseID[notificationTag](raw, prefixNotification)
}
