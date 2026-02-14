package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	notifdomain "github.com/0xsj/canopy-backend/internal/notification/domain"
	notifservice "github.com/0xsj/canopy-backend/internal/notification/service"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// ── Mocks ───────────────────────────────────────────────────────

type mockNotificationRepo struct {
	created []notifdomain.Notification
}

func (m *mockNotificationRepo) Create(_ context.Context, n notifdomain.Notification) error {
	m.created = append(m.created, n)
	return nil
}
func (m *mockNotificationRepo) FindByID(_ context.Context, _ notifdomain.NotificationID) (notifdomain.Notification, error) {
	return notifdomain.Notification{}, nil
}
func (m *mockNotificationRepo) FindByUser(_ context.Context, _ types.UserID, _ int) ([]notifdomain.Notification, error) {
	return nil, nil
}
func (m *mockNotificationRepo) FindUnreadByUser(_ context.Context, _ types.UserID) ([]notifdomain.Notification, error) {
	return nil, nil
}
func (m *mockNotificationRepo) Update(_ context.Context, _ notifdomain.Notification) error {
	return nil
}
func (m *mockNotificationRepo) MarkAllRead(_ context.Context, _ types.UserID) error {
	return nil
}

type mockSubscriptionRepo struct{}

func (m *mockSubscriptionRepo) Save(_ context.Context, _ notifdomain.Subscription) error {
	return nil
}
func (m *mockSubscriptionRepo) FindByUserAndWorkspace(_ context.Context, _ types.UserID, _ types.WorkspaceID) (notifdomain.Subscription, error) {
	return notifdomain.Subscription{}, nil
}
func (m *mockSubscriptionRepo) FindByWorkspace(_ context.Context, _ types.WorkspaceID) ([]notifdomain.Subscription, error) {
	return nil, nil
}

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ context.Context, _ events.Event) error { return nil }

type mockMemberLister struct {
	members []wsdomain.WorkspaceMember
	err     error
}

func (m *mockMemberLister) FindByWorkspace(_ context.Context, _ types.WorkspaceID) ([]wsdomain.WorkspaceMember, error) {
	return m.members, m.err
}

// ── Helpers ─────────────────────────────────────────────────────

func newTestNotifService(repo *mockNotificationRepo) *notifservice.Service {
	return notifservice.New(repo, &mockSubscriptionRepo{}, nil, &mockPublisher{}, logger.NewNoop())
}

func testEvent(eventType, workspaceID string, data any) events.Event {
	raw, _ := json.Marshal(data)
	return events.Event{
		ID:          "evt_test",
		Type:        eventType,
		Subject:     "workspace." + workspaceID + ".test." + eventType,
		WorkspaceID: workspaceID,
		OccurredAt:  time.Now().UTC(),
		Data:        raw,
	}
}

func testMembers(wsID types.WorkspaceID, userIDs ...string) []wsdomain.WorkspaceMember {
	members := make([]wsdomain.WorkspaceMember, len(userIDs))
	for i, uid := range userIDs {
		m, _ := wsdomain.NewWorkspaceMember(wsID, types.UserIDFrom(uid), wsdomain.RoleParticipant)
		members[i] = m
	}
	return members
}

// ── Tests ───────────────────────────────────────────────────────

func TestNotificationHandler_MappedEvent_NotifiesMembers(t *testing.T) {
	repo := &mockNotificationRepo{}
	svc := newTestNotifService(repo)
	wsID := types.WorkspaceIDFrom("ws_abc")
	members := &mockMemberLister{members: testMembers(wsID, "user_alice", "user_bob", "user_carol")}

	handler := notificationHandler(svc, members, logger.NewNoop())

	evt := testEvent("exploration.leaf.created", "ws_abc", map[string]string{
		"leaf_id":      "leaf_123",
		"workspace_id": "ws_abc",
		"author_id":    "user_alice",
		"title":        "My Leaf",
	})

	err := handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Alice is the author — should be excluded. Bob and Carol get notified.
	if got := len(repo.created); got != 2 {
		t.Fatalf("expected 2 notifications, got %d", got)
	}

	userIDs := make(map[string]bool)
	for _, n := range repo.created {
		userIDs[n.UserID().String()] = true
		if n.Title() != "New leaf created" {
			t.Errorf("unexpected title: %q", n.Title())
		}
		if n.ResourceType() != "leaf" {
			t.Errorf("unexpected resource type: %q", n.ResourceType())
		}
		if n.Channel() != notifdomain.ChannelInApp {
			t.Errorf("unexpected channel: %q", n.Channel())
		}
	}
	if userIDs["user_alice"] {
		t.Error("actor (alice) should not receive notification")
	}
	if !userIDs["user_bob"] || !userIDs["user_carol"] {
		t.Error("bob and carol should receive notifications")
	}
}

func TestNotificationHandler_NoActorField_NotifiesAll(t *testing.T) {
	repo := &mockNotificationRepo{}
	svc := newTestNotifService(repo)
	wsID := types.WorkspaceIDFrom("ws_abc")
	members := &mockMemberLister{members: testMembers(wsID, "user_alice", "user_bob")}

	handler := notificationHandler(svc, members, logger.NewNoop())

	// convergence.consensus.reached has no actor field — all members notified.
	evt := testEvent("convergence.consensus.reached", "ws_abc", map[string]string{
		"checkpoint_id": "cp_123",
		"workspace_id":  "ws_abc",
	})

	err := handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if got := len(repo.created); got != 2 {
		t.Fatalf("expected 2 notifications (all members), got %d", got)
	}
}

func TestNotificationHandler_UnmappedEvent_Acks(t *testing.T) {
	repo := &mockNotificationRepo{}
	svc := newTestNotifService(repo)
	members := &mockMemberLister{}

	handler := notificationHandler(svc, members, logger.NewNoop())

	evt := testEvent("identity.user.registered", "ws_abc", map[string]string{
		"user_id": "user_alice",
	})

	err := handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if got := len(repo.created); got != 0 {
		t.Fatalf("expected 0 notifications for unmapped event, got %d", got)
	}
}

func TestNotificationHandler_EmptyWorkspaceID_Acks(t *testing.T) {
	repo := &mockNotificationRepo{}
	svc := newTestNotifService(repo)
	members := &mockMemberLister{}

	handler := notificationHandler(svc, members, logger.NewNoop())

	evt := testEvent("exploration.leaf.created", "", map[string]string{
		"leaf_id":   "leaf_123",
		"author_id": "user_alice",
	})

	err := handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if got := len(repo.created); got != 0 {
		t.Fatalf("expected 0 notifications for empty workspace, got %d", got)
	}
}

func TestNotificationHandler_MemberLookupFailure_Acks(t *testing.T) {
	repo := &mockNotificationRepo{}
	svc := newTestNotifService(repo)
	members := &mockMemberLister{err: errors.New("db connection lost")}

	handler := notificationHandler(svc, members, logger.NewNoop())

	evt := testEvent("seed.planted", "ws_abc", map[string]string{
		"seed_id":      "seed_123",
		"workspace_id": "ws_abc",
		"author_id":    "user_alice",
	})

	err := handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("handler should ack on member lookup failure, got error: %v", err)
	}

	if got := len(repo.created); got != 0 {
		t.Fatalf("expected 0 notifications on member lookup failure, got %d", got)
	}
}

func TestNotificationHandler_BadJSON_Acks(t *testing.T) {
	repo := &mockNotificationRepo{}
	svc := newTestNotifService(repo)
	members := &mockMemberLister{}

	handler := notificationHandler(svc, members, logger.NewNoop())

	evt := events.Event{
		ID:          "evt_test",
		Type:        "exploration.leaf.created",
		Subject:     "workspace.ws_abc.exploration.leaf.created",
		WorkspaceID: "ws_abc",
		OccurredAt:  time.Now().UTC(),
		Data:        json.RawMessage(`{invalid json`),
	}

	err := handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("handler should ack on bad JSON, got error: %v", err)
	}
}

func TestNotificationHandler_AllMappedEventTypes(t *testing.T) {
	// Verify every entry in the registry is a known event type
	// and has required fields.
	for eventType, mapping := range notificationRegistry {
		if mapping.title == "" {
			t.Errorf("event %q has empty title", eventType)
		}
		if mapping.resourceType == "" {
			t.Errorf("event %q has empty resource type", eventType)
		}
		if mapping.resourceField == "" {
			t.Errorf("event %q has empty resource field", eventType)
		}
	}
}

func TestNotificationMappedEvents_ReturnsAllKeys(t *testing.T) {
	mapped := notificationMappedEvents()
	if len(mapped) != len(notificationRegistry) {
		t.Errorf("expected %d mapped events, got %d", len(notificationRegistry), len(mapped))
	}
}
