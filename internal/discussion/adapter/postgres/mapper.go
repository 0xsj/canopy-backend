package postgres

import (
	"encoding/json"
	"time"

	"github.com/0xsj/canopy-backend/internal/discussion/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/discussion/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- domain → sqlc params ---

func threadToSaveParams(t domain.Thread) (sqlc.SaveThreadParams, error) {
	commentsJSON, err := marshalComments(t.Comments())
	if err != nil {
		return sqlc.SaveThreadParams{}, err
	}
	return sqlc.SaveThreadParams{
		LeafID:   t.LeafID().String(),
		Comments: commentsJSON,
	}, nil
}

// --- sqlc model → domain ---

func threadToDomain(row sqlc.Thread) (domain.Thread, error) {
	comments, err := unmarshalComments(row.Comments)
	if err != nil {
		return domain.Thread{}, err
	}
	return domain.ReconstructThread(
		types.LeafIDFrom(row.LeafID),
		comments,
	), nil
}

// --- JSONB comment DTO ---

type commentDTO struct {
	AuthorID  string    `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func marshalComments(comments []domain.Comment) ([]byte, error) {
	dtos := make([]commentDTO, len(comments))
	for i, c := range comments {
		dtos[i] = commentDTO{
			AuthorID:  c.AuthorID().String(),
			Content:   c.Content(),
			CreatedAt: c.CreatedAt().Time(),
		}
	}
	return json.Marshal(dtos)
}

func unmarshalComments(data []byte) ([]domain.Comment, error) {
	var dtos []commentDTO
	if err := json.Unmarshal(data, &dtos); err != nil {
		return nil, err
	}
	comments := make([]domain.Comment, len(dtos))
	for i, dto := range dtos {
		comments[i] = domain.ReconstructComment(
			types.UserIDFrom(dto.AuthorID),
			dto.Content,
			types.TimestampFrom(dto.CreatedAt),
		)
	}
	return comments, nil
}
