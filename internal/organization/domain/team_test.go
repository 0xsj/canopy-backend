package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewTeam_ValidInput(t *testing.T) {
	orgID := types.NewOrgID()
	team, err := NewTeam(orgID, "Engineering")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if team.ID().IsZero() {
		t.Fatal("expected non-zero ID")
	}
	if team.OrgID() != orgID {
		t.Errorf("org ID = %v, want %v", team.OrgID(), orgID)
	}
	if team.Name() != "Engineering" {
		t.Errorf("name = %q, want %q", team.Name(), "Engineering")
	}
}

func TestNewTeam_RejectsEmptyName(t *testing.T) {
	_, err := NewTeam(types.NewOrgID(), "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestTeamUpdateDetails_ChangesNameAndDescription(t *testing.T) {
	team, _ := NewTeam(types.NewOrgID(), "Engineering")

	if err := team.UpdateDetails("Platform", "Platform engineering team"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if team.Name() != "Platform" {
		t.Errorf("name = %q, want %q", team.Name(), "Platform")
	}
	if team.Description() != "Platform engineering team" {
		t.Errorf("description = %q, want %q", team.Description(), "Platform engineering team")
	}
}

func TestNewTeamMember_ValidInput(t *testing.T) {
	teamID := types.NewTeamID()
	userID := types.NewUserID()
	m, err := NewTeamMember(teamID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.TeamID() != teamID {
		t.Errorf("team ID = %v, want %v", m.TeamID(), teamID)
	}
	if m.UserID() != userID {
		t.Errorf("user ID = %v, want %v", m.UserID(), userID)
	}
}

func TestNewTeamMember_RejectsZeroTeamID(t *testing.T) {
	_, err := NewTeamMember(types.TeamID{}, types.NewUserID())
	if err == nil {
		t.Fatal("expected error for zero team ID")
	}
}
