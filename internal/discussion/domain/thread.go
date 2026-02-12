package domain

import (
	"fmt"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Comment is a single entry in a discussion thread. Comments are
// append-only — they cannot be edited or deleted.
type Comment struct {
	authorID  types.UserID
	content   string
	createdAt types.Timestamp
}

// NewComment creates a new comment with validated content.
func NewComment(authorID types.UserID, content string) (Comment, error) {
	if authorID.IsZero() {
		return Comment{}, fmt.Errorf("discussion: author ID is required")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return Comment{}, fmt.Errorf("discussion: comment content is required")
	}

	return Comment{
		authorID:  authorID,
		content:   content,
		createdAt: types.Now(),
	}, nil
}

// ReconstructComment builds a Comment from trusted data.
func ReconstructComment(authorID types.UserID, content string, createdAt types.Timestamp) Comment {
	return Comment{
		authorID:  authorID,
		content:   content,
		createdAt: createdAt,
	}
}

func (c Comment) AuthorID() types.UserID     { return c.authorID }
func (c Comment) Content() string            { return c.content }
func (c Comment) CreatedAt() types.Timestamp { return c.createdAt }

// Thread is an ephemeral discussion attached to a leaf. Threads are
// explicitly non-structural — they cannot be promoted, synthesized,
// or referenced in deliverables. Comments are append-only.
type Thread struct {
	leafID   types.LeafID
	comments []Comment
}

// NewThread creates an empty thread for a leaf.
func NewThread(leafID types.LeafID) (Thread, error) {
	if leafID.IsZero() {
		return Thread{}, fmt.Errorf("discussion: leaf ID is required")
	}

	return Thread{
		leafID:   leafID,
		comments: nil,
	}, nil
}

// ReconstructThread builds a Thread from trusted data.
func ReconstructThread(leafID types.LeafID, comments []Comment) Thread {
	return Thread{
		leafID:   leafID,
		comments: comments,
	}
}

// AddComment appends a comment to the thread.
func (t *Thread) AddComment(comment Comment) {
	t.comments = append(t.comments, comment)
}

// CommentCount returns the number of comments in the thread.
func (t Thread) CommentCount() int { return len(t.comments) }

func (t Thread) LeafID() types.LeafID { return t.leafID }
func (t Thread) Comments() []Comment  { return t.comments }
