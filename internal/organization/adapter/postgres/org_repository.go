package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// OrgRepository implements domain.OrgRepository using Postgres.
type OrgRepository struct {
	db database.DBTX
}

// NewOrgRepository creates a new OrgRepository.
func NewOrgRepository(db database.DBTX) *OrgRepository {
	return &OrgRepository{db: db}
}

var _ domain.OrgRepository = (*OrgRepository)(nil)

func (r *OrgRepository) Create(ctx context.Context, org domain.Organization) error {
	const op = "organization: create org"
	const query = `
		INSERT INTO organizations (id, name, slug, personal, owner_id, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	settingsJSON, err := json.Marshal(org.Settings())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	ts := org.Timestamps()
	_, err = r.db.Exec(ctx, query,
		org.ID().String(),
		org.Name(),
		org.Slug(),
		org.IsPersonal(),
		org.OwnerID().String(),
		settingsJSON,
		ts.CreatedAt.Time(),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *OrgRepository) FindByID(ctx context.Context, id types.OrgID) (domain.Organization, error) {
	const op = "organization: find org by id"
	const query = `
		SELECT id, name, slug, personal, owner_id, settings, created_at, updated_at
		FROM organizations WHERE id = $1`

	return scanOrg(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *OrgRepository) FindBySlug(ctx context.Context, slug string) (domain.Organization, error) {
	const op = "organization: find org by slug"
	const query = `
		SELECT id, name, slug, personal, owner_id, settings, created_at, updated_at
		FROM organizations WHERE slug = $1`

	return scanOrg(r.db.QueryRow(ctx, query, slug), op)
}

func (r *OrgRepository) FindPersonalByOwner(ctx context.Context, ownerID types.UserID) (domain.Organization, error) {
	const op = "organization: find personal org by owner"
	const query = `
		SELECT id, name, slug, personal, owner_id, settings, created_at, updated_at
		FROM organizations WHERE owner_id = $1 AND personal = true`

	return scanOrg(r.db.QueryRow(ctx, query, ownerID.String()), op)
}

func (r *OrgRepository) Update(ctx context.Context, org domain.Organization) error {
	const op = "organization: update org"
	const query = `
		UPDATE organizations
		SET name = $2, slug = $3, settings = $4, updated_at = $5
		WHERE id = $1`

	settingsJSON, err := json.Marshal(org.Settings())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	ts := org.Timestamps()
	tag, err := r.db.Exec(ctx, query,
		org.ID().String(),
		org.Name(),
		org.Slug(),
		settingsJSON,
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

// --- helpers ---

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOrg(row rowScanner, op string) (domain.Organization, error) {
	var (
		rawID        string
		name         string
		slug         string
		personal     bool
		rawOwnerID   string
		settingsJSON []byte
		createdAt    time.Time
		updatedAt    time.Time
	)

	if err := row.Scan(&rawID, &name, &slug, &personal, &rawOwnerID, &settingsJSON, &createdAt, &updatedAt); err != nil {
		return domain.Organization{}, database.MapQueryError(err, op)
	}

	settings := make(map[string]any)
	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &settings); err != nil {
			return domain.Organization{}, database.MapQueryError(err, op)
		}
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructOrganization(
		types.OrgIDFrom(rawID),
		name,
		slug,
		personal,
		types.UserIDFrom(rawOwnerID),
		settings,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(createdAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func tsUpdatedAt(ts types.Timestamps) time.Time {
	if ts.UpdatedAt != nil {
		return ts.UpdatedAt.Time()
	}
	return ts.CreatedAt.Time()
}
