package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewThread_ValidInput(t *testing.T) {
	thread, err := NewThread(types.NewLeafID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if thread.CommentCount() != 0 {
		t.Errorf("comment count = %d, want 0", thread.CommentCount())
	}
}

func TestNewComment_ValidInput(t *testing.T) {
	c, err := NewComment(types.NewUserID(), "Great insight!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.Content() != "Great insight!" {
		t.Errorf("content = %q, want %q", c.Content(), "Great insight!")
	}
}

func TestNewComment_RejectsEmptyContent(t *testing.T) {
	_, err := NewComment(types.NewUserID(), "")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestThread_AddComment(t *testing.T) {
	thread, _ := NewThread(types.NewLeafID())
	c1, _ := NewComment(types.NewUserID(), "First")
	c2, _ := NewComment(types.NewUserID(), "Second")

	thread.AddComment(c1)
	thread.AddComment(c2)

	if thread.CommentCount() != 2 {
		t.Errorf("comment count = %d, want 2", thread.CommentCount())
	}
}
