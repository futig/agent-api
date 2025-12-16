CREATE TABLE IF NOT EXISTS iteration_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    iteration_id UUID NOT NULL REFERENCES session_iterations(id) ON DELETE CASCADE,
    question_number INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    question TEXT NOT NULL,
    explanation TEXT NOT NULL,
    answer TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    answered_at TIMESTAMP
);

CREATE INDEX idx_questions_iteration_id ON iteration_questions(iteration_id);
