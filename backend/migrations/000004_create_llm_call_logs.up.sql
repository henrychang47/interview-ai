CREATE TABLE llm_call_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    interview_id UUID REFERENCES interviews(id) ON DELETE SET NULL,
    question_id UUID REFERENCES questions(id) ON DELETE SET NULL,
    answer_id UUID REFERENCES answers(id) ON DELETE SET NULL,
    status TEXT NOT NULL,
    latency_ms INTEGER,
    input_tokens INTEGER,
    output_tokens INTEGER,
    total_tokens INTEGER,
    error_code TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_llm_call_logs_created_at ON llm_call_logs(created_at);
CREATE INDEX idx_llm_call_logs_operation ON llm_call_logs(operation);
CREATE INDEX idx_llm_call_logs_interview_id ON llm_call_logs(interview_id);
CREATE INDEX idx_llm_call_logs_question_id ON llm_call_logs(question_id);
CREATE INDEX idx_llm_call_logs_answer_id ON llm_call_logs(answer_id);
