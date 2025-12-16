-- Telegram bot user state tracking
CREATE TABLE telegram_sessions (
    user_id BIGINT PRIMARY KEY,
    session_id UUID REFERENCES sessions(id) ON DELETE CASCADE,
    state_data JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_telegram_sessions_session_id ON telegram_sessions(session_id);
CREATE INDEX idx_telegram_sessions_updated_at ON telegram_sessions(updated_at);

-- Telegram user profiles (optional, for personalization)
CREATE TABLE telegram_users (
    user_id BIGINT PRIMARY KEY,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    language_code VARCHAR(10) DEFAULT 'ru',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_telegram_users_last_active ON telegram_users(last_active_at DESC);
