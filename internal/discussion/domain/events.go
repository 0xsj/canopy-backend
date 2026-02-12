package domain

import "time"

// Event subjects published by the discussion context.
const (
	SubjectThreadCommentAdded = "discussion.comment.added"
)

// ThreadCommentAddedData is published when a comment is added to a thread.
type ThreadCommentAddedData struct {
	LeafID      string    `json:"leaf_id"`
	AuthorID    string    `json:"author_id"`
	CommentText string    `json:"comment_text"`
	Timestamp   time.Time `json:"timestamp"`
}
