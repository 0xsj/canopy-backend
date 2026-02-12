package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// DigestFrequency determines how often a user receives digest notifications.
type DigestFrequency string

const (
	DigestDaily      DigestFrequency = "daily"
	DigestPerSession DigestFrequency = "per_session"
	DigestWeekly     DigestFrequency = "weekly"
)

// IsValid reports whether the frequency is a recognized value.
func (f DigestFrequency) IsValid() bool {
	switch f {
	case DigestDaily, DigestPerSession, DigestWeekly:
		return true
	}
	return false
}

// Subscription defines a user's notification preferences for a workspace.
// Each user has one subscription per workspace, controlling which event
// types trigger notifications and through which channels.
type Subscription struct {
	userID          types.UserID
	workspaceID     types.WorkspaceID
	channels        []Channel // which channels to use
	digestFrequency DigestFrequency
	timestamps      types.Timestamps
}

// NewSubscription creates a new notification subscription with defaults.
func NewSubscription(userID types.UserID, workspaceID types.WorkspaceID) (Subscription, error) {
	if userID.IsZero() {
		return Subscription{}, fmt.Errorf("notification: user ID is required")
	}
	if workspaceID.IsZero() {
		return Subscription{}, fmt.Errorf("notification: workspace ID is required")
	}

	return Subscription{
		userID:          userID,
		workspaceID:     workspaceID,
		channels:        []Channel{ChannelInApp},
		digestFrequency: DigestDaily,
		timestamps:      types.NewMutableTimestamps(),
	}, nil
}

// ReconstructSubscription builds a Subscription from trusted data.
func ReconstructSubscription(
	userID types.UserID,
	workspaceID types.WorkspaceID,
	channels []Channel,
	digestFrequency DigestFrequency,
	timestamps types.Timestamps,
) Subscription {
	return Subscription{
		userID:          userID,
		workspaceID:     workspaceID,
		channels:        channels,
		digestFrequency: digestFrequency,
		timestamps:      timestamps,
	}
}

// UpdateChannels replaces the notification channels.
func (s *Subscription) UpdateChannels(channels []Channel) error {
	if len(channels) == 0 {
		return fmt.Errorf("notification: at least one channel is required")
	}
	for _, ch := range channels {
		if !ch.IsValid() {
			return fmt.Errorf("notification: invalid channel %q", ch)
		}
	}
	s.channels = channels
	s.timestamps.Touch()
	return nil
}

// UpdateDigestFrequency changes the digest delivery schedule.
func (s *Subscription) UpdateDigestFrequency(freq DigestFrequency) error {
	if !freq.IsValid() {
		return fmt.Errorf("notification: invalid digest frequency %q", freq)
	}
	s.digestFrequency = freq
	s.timestamps.Touch()
	return nil
}

func (s Subscription) UserID() types.UserID             { return s.userID }
func (s Subscription) WorkspaceID() types.WorkspaceID   { return s.workspaceID }
func (s Subscription) Channels() []Channel              { return s.channels }
func (s Subscription) DigestFrequency() DigestFrequency { return s.digestFrequency }
func (s Subscription) Timestamps() types.Timestamps     { return s.timestamps }
