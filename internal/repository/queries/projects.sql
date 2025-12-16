-- name: CreateProject :one
INSERT INTO projects (id, title, description, created_at)
VALUES ($1, $2, $3, NOW())
RETURNING *;

-- name: GetProject :one
SELECT *
FROM projects
WHERE id = $1;

-- name: ListProjects :many
SELECT *
FROM projects
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;
