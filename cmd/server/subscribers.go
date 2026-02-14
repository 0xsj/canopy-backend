package main

import (
	"context"
	"encoding/json"

	"github.com/0xsj/canopy-backend/internal/ledger/domain"
	ledgerservice "github.com/0xsj/canopy-backend/internal/ledger/service"
	notifservice "github.com/0xsj/canopy-backend/internal/notification/service"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// registerSubscribers sets up durable JetStream consumers for audit and notifications.
// Returns a cleanup function that unsubscribes all consumers.
func registerSubscribers(
	ctx context.Context,
	sub events.Subscriber,
	ledger *ledgerservice.Service,
	notif *notifservice.Service,
	members workspaceMemberLister,
	resolver *llmProviderResolver,
	log logger.Logger,
) (func(), error) {
	var subs []events.Subscription

	// Ledger: system audit (all events).
	systemSub, err := sub.Subscribe(ctx, "workspace.>", systemAuditHandler(ledger, log), events.WithConsumer("ledger_system_audit"))
	if err != nil {
		return nil, err
	}
	subs = append(subs, systemSub)

	// Ledger: domain audit (selective events).
	domainSub, err := sub.Subscribe(ctx, "workspace.>", domainAuditHandler(ledger, log), events.WithConsumer("ledger_domain_audit"))
	if err != nil {
		cleanupAll(subs)
		return nil, err
	}
	subs = append(subs, domainSub)

	// Notification: in-app notifications for workspace members.
	notifSub, err := sub.Subscribe(ctx, "workspace.>", notificationHandler(notif, members, log), events.WithConsumer("notification_events"))
	if err != nil {
		cleanupAll(subs)
		return nil, err
	}
	subs = append(subs, notifSub)

	// Workspace LLM config cache invalidation.
	llmSub, err := sub.Subscribe(ctx, "workspace.*.workspace.workspace.llm_config.*",
		llmConfigHandler(resolver, log), events.WithConsumer("llm_config_cache"))
	if err != nil {
		cleanupAll(subs)
		return nil, err
	}
	subs = append(subs, llmSub)

	// User LLM config cache invalidation.
	userLLMSub, err := sub.Subscribe(ctx, "workspace.>",
		userLLMConfigHandler(resolver, log), events.WithConsumer("user_llm_config_cache"))
	if err != nil {
		cleanupAll(subs)
		return nil, err
	}
	subs = append(subs, userLLMSub)

	log.Info("event subscribers registered",
		logger.Int("count", len(subs)),
	)

	return func() { cleanupAll(subs) }, nil
}

// systemAuditHandler records every event as a system-level audit entry.
func systemAuditHandler(ledger *ledgerservice.Service, log logger.Logger) events.Handler {
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
	action        domain.Action
	resourceType  string
	actorField    string // JSON field for actor ID; "" means use "system"
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
	"seed.constraints.updated": {
		action:        domain.ActionUpdated,
		resourceType:  "seed",
		actorField:    "",
		resourceField: "seed_id",
	},
	"workspace.config.updated": {
		action:        domain.ActionUpdated,
		resourceType:  "workspace",
		actorField:    "",
		resourceField: "workspace_id",
	},
	"workspace.phase.transitioned": {
		action:        domain.ActionUpdated,
		resourceType:  "workspace",
		actorField:    "",
		resourceField: "workspace_id",
	},
	"exploration.connection.created": {
		action:        domain.ActionCreated,
		resourceType:  "connection",
		actorField:    "author_id",
		resourceField: "connection_id",
	},
	"exploration.leaf.promoted": {
		action:        domain.ActionPromoted,
		resourceType:  "leaf",
		actorField:    "",
		resourceField: "leaf_id",
	},
	"discussion.comment.added": {
		action:        domain.ActionCreated,
		resourceType:  "comment",
		actorField:    "author_id",
		resourceField: "leaf_id",
	},
	"session.started": {
		action:        domain.ActionCreated,
		resourceType:  "session",
		actorField:    "user_id",
		resourceField: "session_id",
	},
	"convergence.checkpoint.created": {
		action:        domain.ActionCreated,
		resourceType:  "checkpoint",
		actorField:    "",
		resourceField: "checkpoint_id",
	},
	"convergence.consensus.reached": {
		action:        domain.ActionPromoted,
		resourceType:  "checkpoint",
		actorField:    "",
		resourceField: "checkpoint_id",
	},
	"synthesis.started": {
		action:        domain.ActionCreated,
		resourceType:  "synthesis",
		actorField:    "initiator_id",
		resourceField: "synthesis_id",
	},
	"synthesis.completed": {
		action:        domain.ActionUpdated,
		resourceType:  "synthesis",
		actorField:    "",
		resourceField: "synthesis_id",
	},
	"deliverable.draft.created": {
		action:        domain.ActionCreated,
		resourceType:  "deliverable",
		actorField:    "",
		resourceField: "deliverable_id",
	},
	"deliverable.updated": {
		action:        domain.ActionUpdated,
		resourceType:  "deliverable",
		actorField:    "",
		resourceField: "deliverable_id",
	},
	"workspace.llm_config.updated": {
		action:        domain.ActionUpdated,
		resourceType:  "llm_config",
		actorField:    "",
		resourceField: "workspace_id",
	},
	"workspace.llm_config.deleted": {
		action:        domain.ActionDeleted,
		resourceType:  "llm_config",
		actorField:    "",
		resourceField: "workspace_id",
	},
	"identity.user.llm_config.updated": {
		action:        domain.ActionUpdated,
		resourceType:  "user_llm_config",
		actorField:    "user_id",
		resourceField: "user_id",
	},
	"identity.user.llm_config.deleted": {
		action:        domain.ActionDeleted,
		resourceType:  "user_llm_config",
		actorField:    "user_id",
		resourceField: "user_id",
	},
}

// domainAuditHandler records high-value events as semantic domain audit entries.
func domainAuditHandler(ledger *ledgerservice.Service, log logger.Logger) events.Handler {
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

// llmConfigHandler invalidates the cached LLM provider when a workspace's
// LLM config is updated or deleted. This ensures the resolver re-reads
// the config from the database on the next request.
func llmConfigHandler(resolver *llmProviderResolver, log logger.Logger) events.Handler {
	return func(_ context.Context, event events.Event) error {
		wsID := event.WorkspaceID
		if wsID == "" {
			var data struct {
				WorkspaceID string `json:"workspace_id"`
			}
			if err := event.Decode(&data); err == nil {
				wsID = data.WorkspaceID
			}
		}

		if wsID == "" {
			log.Warn("llm config handler: no workspace_id", logger.String("type", event.Type))
			return nil
		}

		resolver.InvalidateCache(wsID)
		log.Debug("llm provider cache invalidated",
			logger.String("workspace_id", wsID),
			logger.String("event", event.Type),
		)

		return nil
	}
}

// userLLMConfigHandler invalidates the cached LLM provider when a user's
// personal LLM config is updated or deleted. Subscribes to workspace.> and
// filters for identity.user.llm_config.* event types.
func userLLMConfigHandler(resolver *llmProviderResolver, log logger.Logger) events.Handler {
	return func(_ context.Context, event events.Event) error {
		if event.Type != "identity.user.llm_config.updated" && event.Type != "identity.user.llm_config.deleted" {
			return nil // not our event
		}

		var data struct {
			UserID string `json:"user_id"`
		}
		if err := event.Decode(&data); err != nil || data.UserID == "" {
			log.Warn("user llm config handler: missing user_id", logger.String("type", event.Type))
			return nil
		}

		resolver.InvalidateUserCache(data.UserID)
		log.Debug("user llm provider cache invalidated",
			logger.String("user_id", data.UserID),
			logger.String("event", event.Type),
		)

		return nil
	}
}

// cleanupAll unsubscribes all subscriptions.
func cleanupAll(subs []events.Subscription) {
	for _, s := range subs {
		s.Unsubscribe()
	}
}
