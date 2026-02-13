package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/organization/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// OrgRepository implements domain.OrgRepository using Postgres via sqlc.
type OrgRepository struct {
	q *sqlc.Queries
}

// NewOrgRepository creates a new OrgRepository.
func NewOrgRepository(db database.DBTX) *OrgRepository {
	return &OrgRepository{q: sqlc.New(db)}
}

var _ domain.OrgRepository = (*OrgRepository)(nil)

func (r *OrgRepository) Create(ctx context.Context, org domain.Organization) error {
	const op = "organization: create org"
	params, err := orgToCreateParams(org)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return database.MapQueryError(r.q.CreateOrg(ctx, params), op)
}

func (r *OrgRepository) FindByID(ctx context.Context, id types.OrgID) (domain.Organization, error) {
	const op = "organization: find org by id"
	row, err := r.q.FindOrgByID(ctx, id.String())
	if err != nil {
		return domain.Organization{}, database.MapQueryError(err, op)
	}
	o, err := orgToDomain(row)
	if err != nil {
		return domain.Organization{}, database.MapQueryError(err, op)
	}
	return o, nil
}

func (r *OrgRepository) FindBySlug(ctx context.Context, slug string) (domain.Organization, error) {
	const op = "organization: find org by slug"
	row, err := r.q.FindOrgBySlug(ctx, slug)
	if err != nil {
		return domain.Organization{}, database.MapQueryError(err, op)
	}
	o, err := orgToDomain(row)
	if err != nil {
		return domain.Organization{}, database.MapQueryError(err, op)
	}
	return o, nil
}

func (r *OrgRepository) FindPersonalByOwner(ctx context.Context, ownerID types.UserID) (domain.Organization, error) {
	const op = "organization: find personal org by owner"
	row, err := r.q.FindPersonalOrgByOwner(ctx, ownerID.String())
	if err != nil {
		return domain.Organization{}, database.MapQueryError(err, op)
	}
	o, err := orgToDomain(row)
	if err != nil {
		return domain.Organization{}, database.MapQueryError(err, op)
	}
	return o, nil
}

func (r *OrgRepository) Update(ctx context.Context, org domain.Organization) error {
	const op = "organization: update org"
	params, err := orgToUpdateParams(org)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	tag, err := r.q.UpdateOrg(ctx, params)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
