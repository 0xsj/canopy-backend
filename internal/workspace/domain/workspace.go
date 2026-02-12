package domain

import (
	"fmt"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Phase represents the current stage of a workspace's lifecycle.
// Phases progress forward only: floor → understory → canopy → emergent.
type Phase string

const (
	PhaseFloor      Phase = "floor"
	PhaseUnderstory Phase = "understory"
	PhaseCanopy     Phase = "canopy"
	PhaseEmergent   Phase = "emergent"
)

// phaseOrder defines the valid progression. A workspace can only move
// forward in this sequence, never backward.
var phaseOrder = map[Phase]int{
	PhaseFloor:      0,
	PhaseUnderstory: 1,
	PhaseCanopy:     2,
	PhaseEmergent:   3,
}

// IsValid reports whether the phase is a recognized value.
func (p Phase) IsValid() bool {
	_, ok := phaseOrder[p]
	return ok
}

// CanTransitionTo reports whether this phase can advance to the target.
func (p Phase) CanTransitionTo(target Phase) bool {
	current, ok1 := phaseOrder[p]
	next, ok2 := phaseOrder[target]
	return ok1 && ok2 && next == current+1
}

// LoreKeeperMode determines how the lore keeper role is filled.
type LoreKeeperMode string

const (
	LoreKeeperHuman  LoreKeeperMode = "human"
	LoreKeeperAI     LoreKeeperMode = "ai"
	LoreKeeperHybrid LoreKeeperMode = "hybrid"
)

// IsValid reports whether the mode is a recognized value.
func (m LoreKeeperMode) IsValid() bool {
	switch m {
	case LoreKeeperHuman, LoreKeeperAI, LoreKeeperHybrid:
		return true
	}
	return false
}

// Workspace is the top-level container for a collaborative session.
// Every workspace belongs to an organization and is optionally scoped
// to a team within that organization.
type Workspace struct {
	id             types.WorkspaceID
	orgID          types.OrgID
	teamID         types.TeamID // zero value if not team-scoped
	name           string
	description    string
	phase          Phase
	loreKeeperMode LoreKeeperMode
	loreKeeperID   types.UserID // zero value if mode is AI
	configuration  map[string]any
	timestamps     types.Timestamps
}

// NewWorkspace creates a new workspace in the floor phase.
func NewWorkspace(orgID types.OrgID, name, description string) (Workspace, error) {
	if orgID.IsZero() {
		return Workspace{}, fmt.Errorf("workspace: org ID is required")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return Workspace{}, fmt.Errorf("workspace: name is required")
	}

	return Workspace{
		id:             types.NewWorkspaceID(),
		orgID:          orgID,
		name:           name,
		description:    strings.TrimSpace(description),
		phase:          PhaseFloor,
		loreKeeperMode: LoreKeeperHuman,
		configuration:  make(map[string]any),
		timestamps:     types.NewMutableTimestamps(),
	}, nil
}

// ReconstructWorkspace builds a Workspace from trusted data.
func ReconstructWorkspace(
	id types.WorkspaceID,
	orgID types.OrgID,
	teamID types.TeamID,
	name string,
	description string,
	phase Phase,
	loreKeeperMode LoreKeeperMode,
	loreKeeperID types.UserID,
	configuration map[string]any,
	timestamps types.Timestamps,
) Workspace {
	if configuration == nil {
		configuration = make(map[string]any)
	}
	return Workspace{
		id:             id,
		orgID:          orgID,
		teamID:         teamID,
		name:           name,
		description:    description,
		phase:          phase,
		loreKeeperMode: loreKeeperMode,
		loreKeeperID:   loreKeeperID,
		configuration:  configuration,
		timestamps:     timestamps,
	}
}

// AssignTeam scopes the workspace to a team. Can only be set once.
func (w *Workspace) AssignTeam(teamID types.TeamID) error {
	if teamID.IsZero() {
		return fmt.Errorf("workspace: team ID is required")
	}
	if !w.teamID.IsZero() {
		return fmt.Errorf("workspace: team already assigned")
	}
	w.teamID = teamID
	w.timestamps.Touch()
	return nil
}

// TransitionPhase advances the workspace to the next phase.
// Phases can only move forward in order.
func (w *Workspace) TransitionPhase(target Phase) error {
	if !w.phase.CanTransitionTo(target) {
		return fmt.Errorf("workspace: cannot transition from %q to %q", w.phase, target)
	}
	w.phase = target
	w.timestamps.Touch()
	return nil
}

// SetLoreKeeper configures the lore keeper for this workspace.
// When mode is Human or Hybrid, a userID is required.
// When mode is AI, userID is ignored.
func (w *Workspace) SetLoreKeeper(mode LoreKeeperMode, userID types.UserID) error {
	if !mode.IsValid() {
		return fmt.Errorf("workspace: invalid lore keeper mode %q", mode)
	}
	if mode != LoreKeeperAI && userID.IsZero() {
		return fmt.Errorf("workspace: user ID required for %q lore keeper mode", mode)
	}

	w.loreKeeperMode = mode
	if mode == LoreKeeperAI {
		w.loreKeeperID = types.UserID{} // clear user
	} else {
		w.loreKeeperID = userID
	}
	w.timestamps.Touch()
	return nil
}

// UpdateDetails changes the workspace's name and/or description.
func (w *Workspace) UpdateDetails(name, description string) error {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	changed := false

	if name != "" && name != w.name {
		w.name = name
		changed = true
	}
	if description != w.description {
		w.description = description
		changed = true
	}

	if !changed {
		return fmt.Errorf("workspace: no changes")
	}

	w.timestamps.Touch()
	return nil
}

// UpdateConfiguration replaces the workspace's configuration map.
func (w *Workspace) UpdateConfiguration(config map[string]any) {
	if config == nil {
		config = make(map[string]any)
	}
	w.configuration = config
	w.timestamps.Touch()
}

func (w Workspace) ID() types.WorkspaceID          { return w.id }
func (w Workspace) OrgID() types.OrgID             { return w.orgID }
func (w Workspace) TeamID() types.TeamID           { return w.teamID }
func (w Workspace) Name() string                   { return w.name }
func (w Workspace) Description() string            { return w.description }
func (w Workspace) Phase() Phase                   { return w.phase }
func (w Workspace) LoreKeeperMode() LoreKeeperMode { return w.loreKeeperMode }
func (w Workspace) LoreKeeperID() types.UserID     { return w.loreKeeperID }
func (w Workspace) Configuration() map[string]any  { return w.configuration }
func (w Workspace) Timestamps() types.Timestamps   { return w.timestamps }
