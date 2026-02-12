package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewNotification_ValidInput(t *testing.T) {
	n, err := NewNotification(
		types.NewUserID(), ChannelInApp, "New leaf", "A new leaf was created",
		"leaf", "leaf_abc", "ws_123",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n.Status() != NotificationPending {
		t.Errorf("status = %q, want %q", n.Status(), NotificationPending)
	}
}

func TestNotification_StatusTransitions(t *testing.T) {
	n, _ := NewNotification(
		types.NewUserID(), ChannelInApp, "Title", "Body",
		"leaf", "leaf_abc", "",
	)

	// pending → delivered
	if err := n.MarkDelivered(); err != nil {
		t.Fatalf("mark delivered: %v", err)
	}
	if n.Status() != NotificationDelivered {
		t.Errorf("status = %q, want %q", n.Status(), NotificationDelivered)
	}

	// delivered → read
	if err := n.MarkRead(); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if n.Status() != NotificationRead {
		t.Errorf("status = %q, want %q", n.Status(), NotificationRead)
	}
}

func TestNotification_RejectsDoubleDelivery(t *testing.T) {
	n, _ := NewNotification(
		types.NewUserID(), ChannelInApp, "Title", "Body",
		"leaf", "leaf_abc", "",
	)
	_ = n.MarkDelivered()

	err := n.MarkDelivered()
	if err == nil {
		t.Fatal("expected error for double delivery")
	}
}

func TestNotification_RejectsDoubleRead(t *testing.T) {
	n, _ := NewNotification(
		types.NewUserID(), ChannelInApp, "Title", "Body",
		"leaf", "leaf_abc", "",
	)
	_ = n.MarkRead()

	err := n.MarkRead()
	if err == nil {
		t.Fatal("expected error for double read")
	}
}

func TestNewSubscription_Defaults(t *testing.T) {
	sub, err := NewSubscription(types.NewUserID(), types.NewWorkspaceID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sub.Channels()) != 1 || sub.Channels()[0] != ChannelInApp {
		t.Errorf("default channels = %v, want [in_app]", sub.Channels())
	}
	if sub.DigestFrequency() != DigestDaily {
		t.Errorf("default frequency = %q, want %q", sub.DigestFrequency(), DigestDaily)
	}
}

func TestSubscription_UpdateChannels(t *testing.T) {
	sub, _ := NewSubscription(types.NewUserID(), types.NewWorkspaceID())

	if err := sub.UpdateChannels([]Channel{ChannelInApp, ChannelPush}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sub.Channels()) != 2 {
		t.Errorf("channels count = %d, want 2", len(sub.Channels()))
	}
}

func TestSubscription_UpdateChannels_RejectsEmpty(t *testing.T) {
	sub, _ := NewSubscription(types.NewUserID(), types.NewWorkspaceID())

	err := sub.UpdateChannels(nil)
	if err == nil {
		t.Fatal("expected error for empty channels")
	}
}
