ALTER TABLE answers
ADD COLUMN analysis_status TEXT NOT NULL DEFAULT 'pending',
ADD COLUMN improvement_suggestions TEXT,
ADD COLUMN analysis_error TEXT,
ADD COLUMN analyzed_at TIMESTAMPTZ;

