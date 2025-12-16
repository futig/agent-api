DROP INDEX IF EXISTS idx_telegram_users_last_active;
DROP TABLE IF EXISTS telegram_users;

DROP INDEX IF EXISTS idx_telegram_sessions_updated_at;
DROP INDEX IF EXISTS idx_telegram_sessions_session_id;
DROP TABLE IF EXISTS telegram_sessions;
