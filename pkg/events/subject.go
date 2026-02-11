package events

import (
	"fmt"
	"strings"
)

// Subject helpers build NATS subjects following the Canopy convention:
//
//	workspace.{workspace_id}.{context}.{event_type}
//
// Examples:
//
//	workspace.ws_abc123.exploration.leaf_created
//	workspace.ws_abc123.convergence.leaf_promoted

// BuildSubject constructs a NATS subject from its components.
//
// For subscriptions, use NATS wildcards:
//   - "*" matches a single token: workspace.*.exploration.leaf_created
//   - ">" matches one or more tokens: workspace.ws_abc123.>
func BuildSubject(workspaceID, context, eventType string) string {
	return fmt.Sprintf("workspace.%s.%s.%s", workspaceID, context, eventType)
}

// ParseSubject extracts components from a NATS subject.
// Returns an error if the subject does not match the expected format.
func ParseSubject(subject string) (workspaceID, context, eventType string, err error) {
	parts := strings.SplitN(subject, ".", 4)
	if len(parts) != 4 || parts[0] != "workspace" {
		return "", "", "", fmt.Errorf("events: invalid subject %q", subject)
	}
	return parts[1], parts[2], parts[3], nil
}
