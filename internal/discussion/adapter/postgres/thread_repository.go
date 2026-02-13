package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/0xsj/canopy-backend/internal/discussion/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// ThreadRepository implements domain.ThreadRepository using Postgres.
type ThreadRepository struct {
	db database.DBTX
}

// NewThreadRepository creates a new ThreadRepository.
func NewThreadRepository(db database.DBTX) *ThreadRepository {
	return &ThreadRepository{db: db}
}

var _ domain.ThreadRepository = (*ThreadRepository)(nil)

func (r *ThreadRepository) Save(ctx context.Context, thread domain.Thread) error {
	const op = "discussion: save thread"
	const query = `
		INSERT INTO threads (leaf_id, comments)
		VALUES ($1, $2)
		ON CONFLICT (leaf_id) DO UPDATE SET comments = $2`

	commentsJSON, err := marshalComments(thread.Comments())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	_, err = r.db.Exec(ctx, query,
		thread.LeafID().String(),
		commentsJSON,
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *ThreadRepository) FindByLeaf(ctx context.Context, leafID types.LeafID) (domain.Thread, error) {
	const op = "discussion: find thread by leaf"
	const query = `SELECT leaf_id, comments FROM threads WHERE leaf_id = $1`

	var (
		rawLeafID    string
		commentsJSON []byte
	)

	if err := r.db.QueryRow(ctx, query, leafID.String()).Scan(&rawLeafID, &commentsJSON); err != nil {
		return domain.Thread{}, database.MapQueryError(err, op)
	}

	comments, err := unmarshalComments(commentsJSON)
	if err != nil {
		return domain.Thread{}, database.MapQueryError(err, op)
	}

	return domain.ReconstructThread(
		types.LeafIDFrom(rawLeafID),
		comments,
	), nil
}

func (r *ThreadRepository) Exists(ctx context.Context, leafID types.LeafID) (bool, error) {
	const op = "discussion: check thread exists"
	const query = `SELECT EXISTS(SELECT 1 FROM threads WHERE leaf_id = $1)`

	var exists bool
	if err := r.db.QueryRow(ctx, query, leafID.String()).Scan(&exists); err != nil {
		return false, database.MapQueryError(err, op)
	}
	return exists, nil
}

// --- comment DTO for JSONB serialization ---

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
