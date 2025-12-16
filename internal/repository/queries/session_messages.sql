-- name: CreateSessionMessage :one
INSERT INTO session_messages (session_id, message_text, created_at)
VALUES ($1, $2, NOW())
RETURNING *;

-- name: GetSessionMessages :many
SELECT *
FROM session_messages
WHERE session_id = $1
ORDER BY created_at ASC;

-- name: DeleteSessionMessages :exec
DELETE FROM session_messages
WHERE session_id = $1;

