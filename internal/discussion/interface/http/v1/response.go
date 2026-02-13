package v1

import (
	"github.com/0xsj/canopy-backend/internal/discussion/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type CommentResponse struct {
	AuthorID  string          `json:"author_id"`
	Content   string          `json:"content"`
	CreatedAt types.Timestamp `json:"created_at"`
}

type ThreadResponse struct {
	LeafID       string            `json:"leaf_id"`
	Comments     []CommentResponse `json:"comments"`
	CommentCount int               `json:"comment_count"`
}

func ThreadFromDomain(t domain.Thread) ThreadResponse {
	comments := make([]CommentResponse, len(t.Comments()))
	for i, c := range t.Comments() {
		comments[i] = CommentResponse{
			AuthorID:  c.AuthorID().String(),
			Content:   c.Content(),
			CreatedAt: c.CreatedAt(),
		}
	}
	return ThreadResponse{
		LeafID:       t.LeafID().String(),
		Comments:     comments,
		CommentCount: t.CommentCount(),
	}
}
