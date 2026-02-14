package main

import (
	"context"
	"encoding/json"

	notifdomain "github.com/0xsj/canopy-backend/internal/notification/domain"
	notifservice "github.com/0xsj/canopy-backend/internal/notification/service"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// workspaceMemberLister is the subset of workspace member repository needed
// by the notification subscriber to resolve target users.
type workspaceMemberLister interface {
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]wsdomain.WorkspaceMember, error)
}

// notificationMapping describes how to turn a domain event into user notifications.
type notificationMapping struct {
	title         string // human-readable title
	body          string // human-readable body (may reference fields)
	resourceType  string // what the notification is about
	resourceField string // JSON field for the resource ID
	actorField    string // JSON field for the actor (excluded from recipients); "" = notify all
}

// notificationRegistry maps event types to their notification mapping.
var notificationRegistry = map[string]notificationMapping{
	// workspace
	"workspace.member.joined": {
		title:         "New member joined",
		body:          "A new member joined the workspace",
		resourceType:  "workspace",
		resourceField: "workspace_id",
		actorField:    "user_id",
	},
	"workspace.phase.transitioned": {
		title:         "Phase transitioned",
		body:          "The workspace moved to a new phase",
		resourceType:  "workspace",
		resourceField: "workspace_id",
		actorField:    "",
	},

	// exploration
	"exploration.leaf.created": {
		title:         "New leaf created",
		body:          "A new leaf was added to the workspace",
		resourceType:  "leaf",
		resourceField: "leaf_id",
		actorField:    "author_id",
	},
	"exploration.branch.created": {
		title:         "New branch started",
		body:          "A new branch was started in the workspace",
		resourceType:  "branch",
		resourceField: "branch_id",
		actorField:    "author_id",
	},

	// discussion
	"discussion.comment.added": {
		title:         "New comment",
		body:          "Someone commented on a leaf",
		resourceType:  "leaf",
		resourceField: "leaf_id",
		actorField:    "author_id",
	},

	// convergence
	"convergence.consensus.reached": {
		title:         "Consensus reached",
		body:          "A checkpoint reached consensus",
		resourceType:  "checkpoint",
		resourceField: "checkpoint_id",
		actorField:    "",
	},
	"convergence.leaf.promoted": {
		title:         "Leaf promoted to canopy",
		body:          "A leaf was promoted to the canopy layer",
		resourceType:  "leaf",
		resourceField: "leaf_id",
		actorField:    "",
	},

	// session
	"session.completed": {
		title:         "Session completed",
		body:          "A thinking session was completed",
		resourceType:  "session",
		resourceField: "session_id",
		actorField:    "user_id",
	},

	// synthesis
	"synthesis.completed": {
		title:         "Synthesis completed",
		body:          "An AI synthesis has finished",
		resourceType:  "synthesis",
		resourceField: "synthesis_id",
		actorField:    "",
	},

	// deliverable
	"deliverable.finalized": {
		title:         "Deliverable finalized",
		body:          "A deliverable has been finalized",
		resourceType:  "deliverable",
		resourceField: "deliverable_id",
		actorField:    "",
	},

	// seed
	"seed.planted": {
		title:         "New seed planted",
		body:          "A new seed was planted in the workspace",
		resourceType:  "seed",
		resourceField: "seed_id",
		actorField:    "author_id",
	},
}

// notificationHandler creates in-app notifications for workspace members
// when high-value domain events occur.
func notificationHandler(
	notif *notifservice.Service,
	members workspaceMemberLister,
	log logger.Logger,
) events.Handler {
	return func(ctx context.Context, event events.Event) error {
		mapping, ok := notificationRegistry[event.Type]
		if !ok {
			return nil // ack — not a mapped event
		}

		workspaceID := event.WorkspaceID
		if workspaceID == "" {
			return nil // ack — org-level events don't target workspace members
		}

		var data map[string]any
		if err := json.Unmarshal(event.Data, &data); err != nil {
			log.Warn("notification: unmarshal failed",
				logger.String("type", event.Type),
				logger.Err(err),
			)
			return nil // ack — won't self-resolve
		}

		resourceID := field(data, mapping.resourceField)

		// Resolve actor to exclude from recipients.
		var actorID string
		if mapping.actorField != "" {
			actorID = field(data, mapping.actorField)
		}

		// Find all workspace members.
		wsID := types.WorkspaceIDFrom(workspaceID)
		wsMembers, err := members.FindByWorkspace(ctx, wsID)
		if err != nil {
			log.Warn("notification: failed to list workspace members",
				logger.String("workspace_id", workspaceID),
				logger.Err(err),
			)
			return nil // ack — transient member lookup failures shouldn't block the queue
		}

		// Send in-app notification to each member, excluding the actor.
		var sendErrors int
		for _, m := range wsMembers {
			userID := m.UserID()
			if actorID != "" && userID.String() == actorID {
				continue
			}

			if err := notif.Send(
				ctx,
				userID,
				notifdomain.ChannelInApp,
				mapping.title,
				mapping.body,
				mapping.resourceType,
				resourceID,
				workspaceID,
			); err != nil {
				sendErrors++
				log.Warn("notification: send failed",
					logger.String("user_id", userID.String()),
					logger.String("type", event.Type),
					logger.Err(err),
				)
			}
		}

		if sendErrors > 0 {
			log.Warn("notification: partial delivery",
				logger.String("type", event.Type),
				logger.Int("failures", sendErrors),
				logger.Int("total", len(wsMembers)),
			)
		}

		return nil // always ack — partial delivery is acceptable
	}
}

// notificationMappedEvents returns the list of event types that trigger notifications.
// Useful for logging at startup.
func notificationMappedEvents() []string {
	out := make([]string, 0, len(notificationRegistry))
	for k := range notificationRegistry {
		out = append(out, k)
	}
	return out
}
