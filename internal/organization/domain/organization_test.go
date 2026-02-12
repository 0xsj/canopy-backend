package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewOrganization_ValidInput(t *testing.T) {
	ownerID := types.NewUserID()
	org, err := NewOrganization("Acme Corp", "acme-corp", ownerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if org.ID().IsZero() {
		t.Fatal("expected non-zero ID")
	}
	if org.Name() != "Acme Corp" {
		t.Errorf("name = %q, want %q", org.Name(), "Acme Corp")
	}
	if org.Slug() != "acme-corp" {
		t.Errorf("slug = %q, want %q", org.Slug(), "acme-corp")
	}
	if org.IsPersonal() {
		t.Error("expected non-personal org")
	}
	if org.OwnerID() != ownerID {
		t.Errorf("owner ID = %v, want %v", org.OwnerID(), ownerID)
	}
}

func TestNewOrganization_RejectsEmptyName(t *testing.T) {
	_, err := NewOrganization("", "acme", types.NewUserID())
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestNewOrganization_RejectsEmptySlug(t *testing.T) {
	_, err := NewOrganization("Acme", "", types.NewUserID())
	if err == nil {
		t.Fatal("expected error for empty slug")
	}
}

func TestNewOrganization_RejectsZeroOwnerID(t *testing.T) {
	_, err := NewOrganization("Acme", "acme", types.UserID{})
	if err == nil {
		t.Fatal("expected error for zero owner ID")
	}
}

func TestNewPersonalOrganization_SetsPersonalFlag(t *testing.T) {
	ownerID := types.NewUserID()
	org := NewPersonalOrganization(ownerID, "Alice")

	if !org.IsPersonal() {
		t.Error("expected personal org")
	}
	if org.OwnerID() != ownerID {
		t.Errorf("owner ID = %v, want %v", org.OwnerID(), ownerID)
	}
	if org.Name() != "Alice's Space" {
		t.Errorf("name = %q, want %q", org.Name(), "Alice's Space")
	}
}

func TestUpdateDetails_ChangesName(t *testing.T) {
	org, _ := NewOrganization("Acme", "acme", types.NewUserID())

	if err := org.UpdateDetails("Acme Inc", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.Name() != "Acme Inc" {
		t.Errorf("name = %q, want %q", org.Name(), "Acme Inc")
	}
}

func TestUpdateDetails_RejectsPersonalOrgRename(t *testing.T) {
	org := NewPersonalOrganization(types.NewUserID(), "Alice")

	err := org.UpdateDetails("New Name", "")
	if err == nil {
		t.Fatal("expected error when renaming personal org")
	}
}

func TestUpdateDetails_RejectsNoChanges(t *testing.T) {
	org, _ := NewOrganization("Acme", "acme", types.NewUserID())

	err := org.UpdateDetails("Acme", "acme")
	if err == nil {
		t.Fatal("expected error when nothing changes")
	}
}
