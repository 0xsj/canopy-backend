-- Add full-text search vector to leaves.
-- Generated column — auto-updates on INSERT, no triggers needed.
-- Weighted: title (A) > summary (B) > key_points (C) > open_questions (D).

ALTER TABLE leaves ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(summary, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(array_to_string(key_points, ' '), '')), 'C') ||
    setweight(to_tsvector('english', coalesce(array_to_string(open_questions, ' '), '')), 'D')
  ) STORED;

CREATE INDEX idx_leaves_search_vector ON leaves USING gin(search_vector);
