CREATE TABLE interview_creation_limits (
    ip_hash TEXT PRIMARY KEY,
    window_started_at TIMESTAMPTZ NOT NULL,
    created_count INTEGER NOT NULL CHECK (created_count >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

