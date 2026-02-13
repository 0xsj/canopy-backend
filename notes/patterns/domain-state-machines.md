# Domain State Machines

## What

Entities with lifecycle phases use string-based enums and explicit transition methods in the domain layer. Transitions are validated by the entity itself — the service layer calls the method and handles the error.

## Why

- Business rules about valid transitions live in the domain, not scattered across services or handlers
- Invalid transitions fail fast with clear error messages
- The entity is always in a consistent state — no external code can set an arbitrary status
- Adding new states or transition rules is localized to one file

## Example

### Workspace Phase Transitions (Unidirectional)

```go
type Phase string

const (
    PhaseFloor      Phase = "floor"
    PhaseUnderstory Phase = "understory"
    PhaseCanopy     Phase = "canopy"
    PhaseEmergent   Phase = "emergent"
)

var phaseOrder = map[Phase]int{
    PhaseFloor: 0, PhaseUnderstory: 1, PhaseCanopy: 2, PhaseEmergent: 3,
}

func (p Phase) CanTransitionTo(target Phase) bool {
    current, ok1 := phaseOrder[p]
    next, ok2 := phaseOrder[target]
    return ok1 && ok2 && next == current+1
}

func (w *Workspace) TransitionPhase(target Phase) error {
    if !w.phase.CanTransitionTo(target) {
        return fmt.Errorf("workspace: cannot transition from %q to %q", w.phase, target)
    }
    w.phase = target
    w.timestamps.Touch()
    return nil
}
```

### Session Status Transitions (Multi-Path)

```go
type SessionStatus string

const (
    StatusActive     SessionStatus = "active"
    StatusCheckpoint SessionStatus = "checkpoint"
    StatusCompleted  SessionStatus = "completed"
    StatusAbandoned  SessionStatus = "abandoned"
)

// active → checkpoint → completed
// active → abandoned
// checkpoint → abandoned

func (s *Session) EnterCheckpoint() error {
    if s.status != StatusActive {
        return fmt.Errorf("session: cannot checkpoint from %q", s.status)
    }
    s.status = StatusCheckpoint
    s.timestamps.Touch()
    return nil
}
```

### Synthesis Workflow (Linear with Failure Branch)

```
pending → processing → completed
pending → processing → failed
```

## Patterns in Use

| Context | Entity | States | Transitions |
|---|---|---|---|
| workspace | Workspace | floor → understory → canopy → emergent | Unidirectional (forward only) |
| session | Session | active → checkpoint → completed / abandoned | Multi-path with terminal states |
| synthesis | SynthesisWorkflow | pending → processing → completed / failed | Linear with failure branch |
| convergence | Checkpoint | open → resolved | Binary |
| notification | Notification | pending → delivered → read | Linear |

## Gotchas

- The service layer should **not** duplicate transition validation — just call the domain method and handle the error
- Transition errors are returned as plain `fmt.Errorf`, not sentinel errors — they're always programming errors or race conditions, not user-facing
- `timestamps.Touch()` is called inside the transition method so the update timestamp is always consistent with state changes
- Reconstructed entities bypass validation — `ReconstructWorkspace(...)` sets any phase directly, trusting the database

## Related

- [[dual-constructor-pattern]] — Reconstruct bypasses state validation
- [[builder-pattern-errors]] — errors use a similar fluent pattern but for construction, not transitions
