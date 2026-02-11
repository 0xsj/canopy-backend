# Config Section Pattern

## What
A decentralized config pattern where each bounded context (or infrastructure concern) declares its own config struct implementing a shared `Section` interface. A central `Loader` composes them, scopes their env var prefixes, and validates them all at startup.

## Why
In a decoupled monolith heading toward microservices, a single global config struct creates coupling. Every context importing from a central config means they all know about each other. The Section pattern keeps config ownership local — each context defines what it needs, the composition root wires them.

## Example
```go
// The interface each section implements.
type Section interface {
    Load(env EnvReader)
    Validate() error
}

// A bounded context defines its own config.
type ExplorationConfig struct {
    MaxLeavesPerBranch int
}

func (c *ExplorationConfig) Load(env config.EnvReader) {
    c.MaxLeavesPerBranch = env.Int("MAX_LEAVES_PER_BRANCH", 50)
}

func (c *ExplorationConfig) Validate() error {
    var v config.Errors
    v.Positive("max_leaves_per_branch", c.MaxLeavesPerBranch)
    return v.Err()
}

// Composition root wires everything.
loader := config.NewLoader("CANOPY")
loader.Register("EXPLORATION", &explorationCfg)
loader.LoadAll() // reads CANOPY_EXPLORATION_MAX_LEAVES_PER_BRANCH
```

When extraction happens, the service just changes the prefix:
```go
loader := config.NewLoader("EXPLORATION") // standalone service
loader.Register("", &cfg) // no section nesting needed
```

## Gotchas
- `Load` should always set sensible defaults — don't rely on zero values being correct.
- `Validate` is called after `Load`, not during. This lets Load complete fully before checking invariants.
- The Errors collector pattern accumulates all failures rather than failing on the first one — better DX at startup.
- Require() returns an error for the caller to handle, not panics — fail-fast means the Loader reports, not individual reads.

## Related
[[functional-options]]
