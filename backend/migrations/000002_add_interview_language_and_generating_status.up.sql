ALTER TABLE interviews
    ADD COLUMN IF NOT EXISTS question_language TEXT NOT NULL DEFAULT 'zh-TW';

ALTER TABLE interviews
    DROP CONSTRAINT IF EXISTS interviews_question_language_check;

ALTER TABLE interviews
    ADD CONSTRAINT interviews_question_language_check
    CHECK (question_language IN ('zh-TW', 'en-US'));

ALTER TABLE interviews
    DROP CONSTRAINT IF EXISTS interviews_status_check;

ALTER TABLE interviews
    ADD CONSTRAINT interviews_status_check
    CHECK (status IN ('created', 'generating_questions', 'questions_ready', 'in_progress', 'completed', 'failed'));
