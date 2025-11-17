
-- Create pgvector extension and issues table

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS issues (
  id TEXT PRIMARY KEY,
  repo TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT,
  labels TEXT[],
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  keywords TEXT[],
  embedding vector(384)
);

CREATE INDEX IF NOT EXISTS idx_issues_repo ON issues(repo);
CREATE INDEX IF NOT EXISTS idx_issues_created ON issues(created_at);
CREATE INDEX IF NOT EXISTS idx_issues_embedding
  ON issues USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);
