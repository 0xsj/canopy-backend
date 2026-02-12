package domain

import (
	"fmt"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// User represents a registered user in the system. The identity context owns
// the user record — who they are, how they authenticate, and their profile
// information. Organization membership and workspace participation are owned
// by their respective contexts.
type User struct {
	id          types.UserID
	externalID  string // OAuth provider ID (e.g., Auth0 sub)
	displayName string
	email       string
	avatarURL   string
	timestamps  types.Timestamps
}

// NewUser creates a new User with a generated ID and validated fields.
// externalID is the OAuth provider's subject identifier.
func NewUser(externalID, displayName, email string) (User, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return User{}, fmt.Errorf("identity: external ID is required")
	}

	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return User{}, fmt.Errorf("identity: display name is required")
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return User{}, fmt.Errorf("identity: email is required")
	}

	return User{
		id:          types.NewUserID(),
		externalID:  externalID,
		displayName: displayName,
		email:       email,
		timestamps:  types.NewMutableTimestamps(),
	}, nil
}

// ReconstructUser builds a User from trusted data (e.g., database row).
// No validation is performed — the caller is responsible for data integrity.
func ReconstructUser(
	id types.UserID,
	externalID string,
	displayName string,
	email string,
	avatarURL string,
	timestamps types.Timestamps,
) User {
	return User{
		id:          id,
		externalID:  externalID,
		displayName: displayName,
		email:       email,
		avatarURL:   avatarURL,
		timestamps:  timestamps,
	}
}

// UpdateProfile applies profile changes. At least one field must differ from
// the current value. Empty strings are ignored (no change).
func (u *User) UpdateProfile(displayName, email, avatarURL string) error {
	displayName = strings.TrimSpace(displayName)
	email = strings.TrimSpace(email)
	avatarURL = strings.TrimSpace(avatarURL)

	changed := false

	if displayName != "" && displayName != u.displayName {
		u.displayName = displayName
		changed = true
	}
	if email != "" && email != u.email {
		u.email = email
		changed = true
	}
	if avatarURL != u.avatarURL {
		u.avatarURL = avatarURL
		changed = true
	}

	if !changed {
		return fmt.Errorf("identity: no profile changes")
	}

	u.timestamps.Touch()
	return nil
}

func (u User) ID() types.UserID             { return u.id }
func (u User) ExternalID() string           { return u.externalID }
func (u User) DisplayName() string          { return u.displayName }
func (u User) Email() string                { return u.email }
func (u User) AvatarURL() string            { return u.avatarURL }
func (u User) Timestamps() types.Timestamps { return u.timestamps }
