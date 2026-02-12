package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// UserRepository defines the persistence port for user entities.
// Implementations live in the adapter layer (e.g., Postgres).
type UserRepository interface {
	// Create persists a new user. Returns an error if the external ID
	// or email already exists.
	Create(ctx context.Context, user User) error

	// FindByID returns a user by internal ID.
	// Returns a NotFound error if no user exists with that ID.
	FindByID(ctx context.Context, id types.UserID) (User, error)

	// FindByExternalID returns a user by OAuth provider subject identifier.
	// Used during the login flow to resolve an authenticated token to a user record.
	// Returns a NotFound error if no user exists with that external ID.
	FindByExternalID(ctx context.Context, externalID string) (User, error)

	// FindByIDs returns users matching the given IDs. The returned slice
	// may be shorter than the input if some IDs do not exist. Order is
	// not guaranteed.
	FindByIDs(ctx context.Context, ids []types.UserID) ([]User, error)

	// Update persists changes to an existing user. Returns a NotFound error
	// if the user does not exist.
	Update(ctx context.Context, user User) error
}
