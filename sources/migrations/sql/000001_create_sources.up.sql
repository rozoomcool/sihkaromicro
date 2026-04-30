-- 000001_create_sources.up.sql
CREATE TABLE sources (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    owner_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'uploading',
    size BIGINT NOT NULL,
    minio_path TEXT,
    job_id TEXT,
    error TEXT,
    source_url TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_sources_owner_id ON sources(owner_id);

CREATE INDEX idx_sources_project_id ON sources(project_id);