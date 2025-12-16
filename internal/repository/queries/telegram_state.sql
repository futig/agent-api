-- name: GetTelegramSession :one
SELECT user_id, session_id, state_data, created_at, updated_at
FROM telegram_sessions
WHERE user_id = $1;

-- name: GetTelegramSessionWithSession :one
SELECT
    ts.user_id,
    ts.session_id,
    ts.state_data,
    ts.created_at as tg_created_at,
    ts.updated_at as tg_updated_at,
    s.id as session_id_full,
    s.status as session_status,
    s.type as session_type,
    s.project_id as session_project_id
FROM telegram_sessions ts
LEFT JOIN sessions s ON ts.session_id = s.id
WHERE ts.user_id = $1;

-- name: GetTelegramSessionBySessionID :one
SELECT user_id, session_id, state_data, created_at, updated_at
FROM telegram_sessions
WHERE session_id = $1;

-- name: UpsertTelegramSession :exec
INSERT INTO telegram_sessions (user_id, session_id, state_data, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET
    session_id = EXCLUDED.session_id,
    state_data = EXCLUDED.state_data,
    updated_at = EXCLUDED.updated_at;

-- name: DeleteTelegramSession :exec
DELETE FROM telegram_sessions
WHERE user_id = $1;
