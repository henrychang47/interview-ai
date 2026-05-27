ALTER TABLE interviews
    DROP CONSTRAINT IF EXISTS interviews_question_language_check;

ALTER TABLE interviews
    DROP COLUMN IF EXISTS question_language;

ALTER TABLE interviews
    DROP CONSTRAINT IF EXISTS interviews_status_check;

ALTER TABLE interviews
    ADD CONSTRAINT interviews_status_check
    CHECK (status IN ('created', 'questions_ready', 'in_progress', 'completed', 'failed'));
