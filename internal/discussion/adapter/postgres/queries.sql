-- name: SaveThread :exec
INSERT INTO threads (leaf_id, comments)
VALUES ($1, $2)
ON CONFLICT (leaf_id) DO UPDATE SET comments = $2;

-- name: FindThreadByLeaf :one
SELECT leaf_id, comments FROM threads WHERE leaf_id = $1;

-- name: ThreadExists :one
SELECT EXISTS(SELECT 1 FROM threads WHERE leaf_id = $1) AS exists;
