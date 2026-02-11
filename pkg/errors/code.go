package errors

// Code is a machine-readable error identifier.
// Clients parse codes to react to specific error conditions.
// Format convention: "{context}_{description}" (e.g., "exploration_leaf_not_found").
type Code string

func (c Code) String() string {
	return string(c)
}
