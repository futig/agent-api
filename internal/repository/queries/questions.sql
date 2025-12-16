-- name: CreateQuestion :one
INSERT INTO iteration_questions (
    id,
    iteration_id,
    question_number,
    status,
    question,
    explanation
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: CreateQuestions :copyfrom
INSERT INTO iteration_questions (
    id,
    iteration_id,
    question_number,
    status,
    question,
    explanation
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: GetQuestionByID :one
SELECT * FROM iteration_questions
WHERE id = $1;

-- name: ListQuestionsByIteration :many
SELECT * FROM iteration_questions
WHERE iteration_id = $1
ORDER BY question_number ASC;

-- name: ListQuestionsBySession :many
SELECT iq.* FROM iteration_questions iq
JOIN session_iterations si ON si.id = iq.iteration_id
WHERE si.session_id = $1
ORDER BY si.iteration_number ASC, iq.question_number ASC;

-- name: UpdateQuestionAnswer :exec
UPDATE iteration_questions
SET answer = $2,
    status = 'ANSWERED',
    answered_at = NOW()
WHERE id = $1;

-- name: SkipQustion :exec
UPDATE iteration_questions
SET status = 'SKIPED'
WHERE id = $1 AND status = 'UNANSWERED';

-- name: GetUnansweredQuestions :many
SELECT iq.* FROM iteration_questions iq
JOIN session_iterations si ON si.id = iq.iteration_id
WHERE si.session_id = $1
  AND (iq.status = 'UNANSWERED' OR iq.status = 'SKIPED')
ORDER BY si.iteration_number ASC, iq.question_number ASC;
