-- name: CreateSession :one
INSERT INTO sessions (
    id,
    status
) VALUES (
    $1, $2
) RETURNING *;

-- name: CreateFilledSession :one
INSERT INTO sessions (
    id,
    project_id,
    status,
    type,
    user_goal,
    project_context
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetSessionByID :one
SELECT * FROM sessions
WHERE id = $1;

-- name: AquireSessionByID :one
UPDATE sessions
SET status = 'Processing', 
    updated_at = NOW()
WHERE id = $1 AND status = 'WaitingForAnswers'
RETURNING *;

-- name: UpdateSessionStatus :one
UPDATE sessions
SET status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSessionRAGProjectContext :one
UPDATE sessions
SET project_context = $1, 
    project_id = $3, 
    updated_at = NOW()
WHERE id = $2
RETURNING *;

-- name: UpdateSessionProjectContext :one
UPDATE sessions
SET project_context = $1, 
    project_id = NULL, 
    updated_at = NOW()
WHERE id = $2
RETURNING *;

-- name: UpdateSessionIteration :one
UPDATE sessions
SET current_iteration = current_iteration + 1,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ResetSessionIteration :one
UPDATE sessions
SET current_iteration = current_iteration - 1,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSessionResult :one
UPDATE sessions
SET status = $2,
    result = $3,
    error = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSessionType :one
UPDATE sessions
SET type = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSessionUserGoal :one
UPDATE sessions
SET user_goal = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;
