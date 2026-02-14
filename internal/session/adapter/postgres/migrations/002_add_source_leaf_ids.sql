ALTER TABLE sessions ADD COLUMN source_leaf_ids TEXT[] NOT NULL DEFAULT '{}';
