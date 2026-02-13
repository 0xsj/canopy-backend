package main

import (
	"context"
	"encoding/json"

	"github.com/0xsj/canopy-backend/internal/ledger/domain"
	"github.com/0xsj/canopy-backend/internal/ledger/service"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// registerSubscribers sets up durable JetStream consumers for the ledger audit trail.
// Returns a cleanup function that unsubscribes all consumers.
func registerSubscribers(ctx context.Context, sub events.Subscriber, ledger *service.Service, log logger.Logger) (func(), error) {
	var subs []events.Subscription

	systemSub, err := sub.Subscribe(ctx, "workspace.>", systemAuditHandler(ledger, log), events.WithConsumer("ledger_system_audit"))
	if err != nil {
		return nil, err
	}
	subs = append(subs, systemSub)

	domainSub, err := sub.Subscribe(ctx, "workspace.>", domainAuditHandler(ledger, log), events.WithConsumer("ledger_domain_audit"))
	if err != nil {
		cleanupAll(subs)
		return nil, err
	}
	subs = append(subs, domainSub)

	log.Info("ledger subscribers registered",
		logger.Int("count", len(subs)),
	)

	return func() { cleanupAll(subs) }, nil
}

// systemAuditHandler records every event as a system-level audit entry.
func systemAuditHandler(ledger *service.Service, log logger.Logger) events.Handler {
	return func(ctx context.Context, event events.Event) error {
		_, sourceCtx, _, err := events.ParseSubject(event.Subject)
		if err != nil {
			log.Warn("system audit: bad subject", logger.String("subject", event.Subject), logger.Err(err))
			return nil // ack — won't self-resolve
		}

		var data map[string]any
		if err := json.Unmarshal(event.Data, &data); err != nil {
			log.Warn("system audit: unmarshal failed", logger.String("type", event.Type), logger.Err(err))
			return nil // ack — won't self-resolve
		}

		if err := ledger.AppendSystem(ctx, event.Subject, data, sourceCtx); err != nil {
			return err // nak — triggers redelivery
		}

		return nil
	}
}

// domainAuditMapping describes how to turn a domain event into a ledger domain entry.
type domainAuditMapping struct {
	action       domain.Action
	resourceType string
	actorField   string // JSON field for actor ID; "" means use "system"
	resourceField string // JSON field for resource ID
}

// domainAuditRegistry maps event types to their audit mapping.
var domainAuditRegistry = map[string]domainAuditMapping{
	"workspace.created": {
		action:        domain.ActionCreated,
		resourceType:  "workspace",
		actorField:    "",
		resourceField: "workspace_id",
	},
	"workspace.member.joined": {
		action:        domain.ActionJoined,
		resourceType:  "member",
		actorField:    "user_id",
		resourceField: "user_id",
	},
	"workspace.member.left": {
		action:        domain.ActionLeft,
		resourceType:  "member",
		actorField:    "user_id",
		resourceField: "user_id",
	},
	"exploration.leaf.created": {
		action:        domain.ActionCreated,
		resourceType:  "leaf",
		actorField:    "author_id",
		resourceField: "leaf_id",
	},
	"exploration.branch.created": {
		action:        domain.ActionCreated,
		resourceType:  "branch",
		actorField:    "author_id",
		resourceField: "branch_id",
	},
	"convergence.leaf.promoted": {
		action:        domain.ActionPromoted,
		resourceType:  "leaf",
		actorField:    "",
		resourceField: "leaf_id",
	},
	"convergence.signal.recorded": {
		action:        domain.ActionSignaled,
		resourceType:  "signal",
		actorField:    "user_id",
		resourceField: "checkpoint_id",
	},
	"session.completed": {
		action:        domain.ActionCreated,
		resourceType:  "session",
		actorField:    "user_id",
		resourceField: "session_id",
	},
	"deliverable.finalized": {
		action:        domain.ActionCreated,
		resourceType:  "deliverable",
		actorField:    "",
		resourceField: "deliverable_id",
	},
	"seed.planted": {
		action:        domain.ActionCreated,
		resourceType:  "seed",
		actorField:    "author_id",
		resourceField: "seed_id",
	},
}

// domainAuditHandler records high-value events as semantic domain audit entries.
func domainAuditHandler(ledger *service.Service, log logger.Logger) events.Handler {
	return func(ctx context.Context, event events.Event) error {
		mapping, ok := domainAuditRegistry[event.Type]
		if !ok {
			return nil // ack — not a mapped event
		}

		var data map[string]any
		if err := json.Unmarshal(event.Data, &data); err != nil {
			log.Warn("domain audit: unmarshal failed", logger.String("type", event.Type), logger.Err(err))
			return nil // ack — won't self-resolve
		}

		actorID := "system"
		if mapping.actorField != "" {
			actorID = field(data, mapping.actorField)
		}

		resourceID := field(data, mapping.resourceField)
		if resourceID == "" {
			log.Warn("domain audit: missing resource ID", logger.String("type", event.Type), logger.String("field", mapping.resourceField))
			return nil // ack — data issue won't self-resolve
		}

		orgID := field(data, "org_id")
		workspaceID := event.WorkspaceID
		if workspaceID == "" {
			workspaceID = field(data, "workspace_id")
		}

		if err := ledger.AppendDomain(ctx, actorID, mapping.action, mapping.resourceType, resourceID, orgID, workspaceID, data); err != nil {
			return err // nak — triggers redelivery
		}

		return nil
	}
}

// field extracts a string value from a decoded JSON map.
func field(data map[string]any, key string) string {
	v, ok := data[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// cleanupAll unsubscribes all subscriptions.
func cleanupAll(subs []events.Subscription) {
	for _, s := range subs {
		s.Unsubscribe()
	}
}
