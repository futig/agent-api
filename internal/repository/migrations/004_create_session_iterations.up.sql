CREATE TABLE IF NOT EXISTS session_iterations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    iteration_number INT NOT NULL,
    title TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_session_iteration UNIQUE (session_id, iteration_number)
);

CREATE INDEX idx_iterations_session_id ON session_iterations(session_id);
CREATE INDEX idx_iterations_session_iteration ON session_iterations(session_id, iteration_number);
