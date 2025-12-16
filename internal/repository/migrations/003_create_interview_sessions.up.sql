CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID,
    status VARCHAR(50) NOT NULL,
    type TEXT,
    user_goal TEXT,
    project_context TEXT,
    current_iteration INT NOT NULL DEFAULT 1,
    result TEXT,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);