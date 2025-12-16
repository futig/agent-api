-- name: AddFile :one
INSERT INTO project_files (id, project_id, filename, size, content_type)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetFiles :many
SELECT *
FROM project_files
WHERE project_id = $1
ORDER BY created_at ASC;

-- name: DeleteProjectFile :exec
DELETE FROM projects WHERE id = $1;
