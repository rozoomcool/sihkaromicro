CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS parent_chunks (
    id         BIGSERIAL PRIMARY KEY,
    source_id  BIGINT    NOT NULL,
    project_id BIGINT    NOT NULL,
    owner_id   TEXT      NOT NULL,
    content    TEXT      NOT NULL,
    metadata   JSONB,
    position   INT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS chunks (
    id              BIGSERIAL PRIMARY KEY,
    parent_chunk_id BIGINT REFERENCES parent_chunks(id) ON DELETE CASCADE,
    source_id       BIGINT    NOT NULL,
    project_id      BIGINT    NOT NULL,
    owner_id        TEXT      NOT NULL,
    content         TEXT      NOT NULL,
    embedding       vector(1536),
    metadata        JSONB,
    position        INT,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS chunks_embedding_idx
    ON chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE INDEX IF NOT EXISTS idx_chunks_owner_project
    ON chunks(owner_id, project_id);

CREATE INDEX IF NOT EXISTS idx_parent_chunks_owner_project
    ON parent_chunks(owner_id, project_id);

CREATE INDEX IF NOT EXISTS idx_chunks_source_id
    ON chunks(source_id);

CREATE INDEX IF NOT EXISTS idx_parent_chunks_source_id
    ON parent_chunks(source_id);
