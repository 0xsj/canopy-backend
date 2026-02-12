package domain

import (
	"fmt"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Organization is the tenancy boundary in Canopy. Billing, data isolation,
// and administrative control are scoped to the organization. Every workspace
// belongs to an organization.
//
// Personal organizations are auto-created on user registration. They are
// invisible in the UI and exist so that solo users have the same data model
// as team users (org_id is always required, never nullable).
type Organization struct {
	id         types.OrgID
	name       string
	slug       string
	personal   bool
	ownerID    types.UserID
	settings   map[string]any
	timestamps types.Timestamps
}

// NewOrganization creates a new organization with validated fields.
func NewOrganization(name, slug string, ownerID types.UserID) (Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Organization{}, fmt.Errorf("organization: name is required")
	}

	slug = strings.TrimSpace(slug)
	if slug == "" {
		return Organization{}, fmt.Errorf("organization: slug is required")
	}

	if ownerID.IsZero() {
		return Organization{}, fmt.Errorf("organization: owner ID is required")
	}

	return Organization{
		id:         types.NewOrgID(),
		name:       name,
		slug:       slug,
		personal:   false,
		ownerID:    ownerID,
		settings:   make(map[string]any),
		timestamps: types.NewMutableTimestamps(),
	}, nil
}

// NewPersonalOrganization creates a personal org for a newly registered user.
// Personal orgs are invisible in the UI and cannot have additional members.
// No validation beyond owner ID — the system controls this creation path.
func NewPersonalOrganization(ownerID types.UserID, displayName string) Organization {
	return Organization{
		id:         types.NewOrgID(),
		name:       displayName + "'s Space",
		slug:       "personal-" + ownerID.String(),
		personal:   true,
		ownerID:    ownerID,
		settings:   make(map[string]any),
		timestamps: types.NewMutableTimestamps(),
	}
}

// ReconstructOrganization builds an Organization from trusted data.
func ReconstructOrganization(
	id types.OrgID,
	name string,
	slug string,
	personal bool,
	ownerID types.UserID,
	settings map[string]any,
	timestamps types.Timestamps,
) Organization {
	if settings == nil {
		settings = make(map[string]any)
	}
	return Organization{
		id:         id,
		name:       name,
		slug:       slug,
		personal:   personal,
		ownerID:    ownerID,
		settings:   settings,
		timestamps: timestamps,
	}
}

// UpdateDetails changes the organization's name and/or slug.
// Empty strings are ignored (no change). Personal orgs cannot be renamed.
func (o *Organization) UpdateDetails(name, slug string) error {
	if o.personal {
		return fmt.Errorf("organization: cannot rename a personal organization")
	}

	name = strings.TrimSpace(name)
	slug = strings.TrimSpace(slug)

	changed := false

	if name != "" && name != o.name {
		o.name = name
		changed = true
	}
	if slug != "" && slug != o.slug {
		o.slug = slug
		changed = true
	}

	if !changed {
		return fmt.Errorf("organization: no changes")
	}

	o.timestamps.Touch()
	return nil
}

// UpdateSettings replaces the organization's settings map.
func (o *Organization) UpdateSettings(settings map[string]any) {
	if settings == nil {
		settings = make(map[string]any)
	}
	o.settings = settings
	o.timestamps.Touch()
}

func (o Organization) ID() types.OrgID              { return o.id }
func (o Organization) Name() string                 { return o.name }
func (o Organization) Slug() string                 { return o.slug }
func (o Organization) IsPersonal() bool             { return o.personal }
func (o Organization) OwnerID() types.UserID        { return o.ownerID }
func (o Organization) Settings() map[string]any     { return o.settings }
func (o Organization) Timestamps() types.Timestamps { return o.timestamps }
