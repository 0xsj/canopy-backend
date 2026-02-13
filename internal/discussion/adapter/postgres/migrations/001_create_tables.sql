CREATE TABLE threads (
    leaf_id  TEXT PRIMARY KEY,
    comments JSONB NOT NULL DEFAULT '[]'
);
