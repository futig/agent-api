package repository

import (
	"context"
	"fmt"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type QuestionRepository interface {
	CreateQuestion(ctx context.Context, question entity.Question) (*entity.Question, error)
	CreateQuestions(ctx context.Context, questions []entity.Question) error
	GetQuestionByID(ctx context.Context, id string) (*entity.Question, error)
	ListQuestionsByIteration(ctx context.Context, iterationID string) ([]*entity.Question, error)
	ListQuestionsBySession(ctx context.Context, sessionID string) ([]*entity.Question, error)
	UpdateQuestionAnswer(ctx context.Context, questionID string, answer string) error
	GetUnansweredQuestions(ctx context.Context, sessionID string) ([]*entity.Question, error)
	SkipQuestion(ctx context.Context, questionID string) error
}

type QuestionPostgres struct {
	queries *sqlc.Queries
	db      *pgxpool.Pool
}

func NewQuestionPostgres(db *pgxpool.Pool) *QuestionPostgres {
	return &QuestionPostgres{
		queries: sqlc.New(db),
		db:      db,
	}
}

// CreateQuestion creates a single question
func (r *QuestionPostgres) CreateQuestion(ctx context.Context, question entity.Question) (*entity.Question, error) {
	questionID, err := uuid.Parse(question.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid question ID: %w", err)
	}

	iterationID, err := uuid.Parse(question.IterationID)
	if err != nil {
		return nil, fmt.Errorf("invalid iteration ID: %w", err)
	}

	dbQuestion, err := r.queries.CreateQuestion(ctx, sqlc.CreateQuestionParams{
		ID: pgtype.UUID{
			Bytes: questionID,
			Valid: true,
		},
		IterationID: pgtype.UUID{
			Bytes: iterationID,
			Valid: true,
		},
		QuestionNumber: int32(question.QuestionNumber),
		Status:         string(question.Status),
		Question:       question.Question,
		Explanation:    question.Explanation,
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to create question", zap.Error(err))
		return nil, err
	}

	return toEntityQuestion(&dbQuestion), nil
}

// CreateQuestions creates multiple questions in a batch
func (r *QuestionPostgres) CreateQuestions(ctx context.Context, questions []entity.Question) error {
	rows := make([][]interface{}, 0, len(questions))

	for _, q := range questions {
		questionID, err := uuid.Parse(q.ID)
		if err != nil {
			return fmt.Errorf("invalid question ID: %w", err)
		}

		iterationID, err := uuid.Parse(q.IterationID)
		if err != nil {
			return fmt.Errorf("invalid iteration ID: %w", err)
		}

		rows = append(rows, []interface{}{
			pgtype.UUID{Bytes: questionID, Valid: true},
			pgtype.UUID{Bytes: iterationID, Valid: true},
			int32(q.QuestionNumber),
			string(q.Status),
			q.Question,
			q.Explanation,
		})
	}

	_, err := r.db.CopyFrom(
		ctx,
		pgx.Identifier{"iteration_questions"},
		[]string{"id", "iteration_id", "question_number", "status", "question", "explanation"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		ctxzap.Error(ctx, "failed to batch create questions", zap.Error(err))
		return err
	}

	return nil
}

// GetQuestionByID retrieves a question by its ID
func (r *QuestionPostgres) GetQuestionByID(ctx context.Context, id string) (*entity.Question, error) {
	questionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid question ID: %w", err)
	}

	dbQuestion, err := r.queries.GetQuestionByID(ctx, pgtype.UUID{
		Bytes: questionID,
		Valid: true,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, entity.ErrQuestionNotFound
		}
		ctxzap.Error(ctx, "failed to get question", zap.Error(err))
		return nil, err
	}

	return toEntityQuestion(&dbQuestion), nil
}

// ListQuestionsByIteration retrieves all questions for an iteration
func (r *QuestionPostgres) ListQuestionsByIteration(ctx context.Context, iterationID string) ([]*entity.Question, error) {
	iterID, err := uuid.Parse(iterationID)
	if err != nil {
		return nil, fmt.Errorf("invalid iteration ID: %w", err)
	}

	dbQuestions, err := r.queries.ListQuestionsByIteration(ctx, pgtype.UUID{
		Bytes: iterID,
		Valid: true,
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to list questions by iteration", zap.Error(err))
		return nil, err
	}

	questions := make([]*entity.Question, 0, len(dbQuestions))
	for _, dbQ := range dbQuestions {
		questions = append(questions, toEntityQuestion(&dbQ))
	}

	return questions, nil
}

// ListQuestionsBySession retrieves all questions for a session across all iterations
func (r *QuestionPostgres) ListQuestionsBySession(ctx context.Context, sessionID string) ([]*entity.Question, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbQuestions, err := r.queries.ListQuestionsBySession(ctx, pgtype.UUID{
		Bytes: sessID,
		Valid: true,
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to list questions by session", zap.Error(err))
		return nil, err
	}

	questions := make([]*entity.Question, 0, len(dbQuestions))
	for _, dbQ := range dbQuestions {
		questions = append(questions, toEntityQuestion(&dbQ))
	}

	return questions, nil
}

// UpdateQuestionAnswer updates a question's answer
func (r *QuestionPostgres) UpdateQuestionAnswer(ctx context.Context, questionID string, answer string) error {
	qID, err := uuid.Parse(questionID)
	if err != nil {
		return fmt.Errorf("invalid question ID: %w", err)
	}

	err = r.queries.UpdateQuestionAnswer(ctx, sqlc.UpdateQuestionAnswerParams{
		ID: pgtype.UUID{
			Bytes: qID,
			Valid: true,
		},
		Answer: pgtype.Text{
			String: answer,
			Valid:  true,
		},
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to update question answer", zap.Error(err))
		return err
	}

	return nil
}

func (r *QuestionPostgres) SkipQuestion(ctx context.Context, questionID string) error {
	qID, err := uuid.Parse(questionID)
	if err != nil {
		return fmt.Errorf("invalid question ID: %w", err)
	}

	err = r.queries.SkipQustion(ctx, pgtype.UUID{
		Bytes: qID,
		Valid: true,
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to update question answer", zap.Error(err))
		return err
	}

	return nil
}

// GetUnansweredQuestions gets all unanswered questions for a session
func (r *QuestionPostgres) GetUnansweredQuestions(ctx context.Context, sessionID string) ([]*entity.Question, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbQuestions, err := r.queries.GetUnansweredQuestions(ctx, pgtype.UUID{
		Bytes: sessID,
		Valid: true,
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to get unanswered questions", zap.Error(err))
		return nil, err
	}

	questions := make([]*entity.Question, 0, len(dbQuestions))
	for _, dbQ := range dbQuestions {
		questions = append(questions, toEntityQuestion(&dbQ))
	}

	return questions, nil
}
