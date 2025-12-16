-- name: CreateIteration :one
INSERT INTO session_iterations (
    id,
    session_id,
    iteration_number,
    title
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: CreateIterations :copyfrom
INSERT INTO session_iterations (
    id,
    session_id,
    iteration_number,
    title
) VALUES (
    $1, $2, $3, $4
);

-- name: GetIterationByID :one
SELECT * FROM session_iterations
WHERE id = $1;

-- name: ListIterationsBySession :many
SELECT * FROM session_iterations
WHERE session_id = $1
ORDER BY iteration_number ASC;

-- name: GetNextIteration :one
SELECT si.* FROM session_iterations as si
JOIN sessions as ss on ss.id = si.session_id
WHERE si.session_id = $1 AND si.iteration_number = ss.current_iteration + 1
LIMIT 1;

-- name: GetCurrentIteration :one
SELECT si.* FROM session_iterations as si
JOIN sessions as ss on ss.id = si.session_id
WHERE si.session_id = $1 AND si.iteration_number = ss.current_iteration
LIMIT 1;
