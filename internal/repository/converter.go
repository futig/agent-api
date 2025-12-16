package repository

import (
	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/repository/sqlc"
	"github.com/google/uuid"
)

func toEntityProject(dbProject *sqlc.Project) *entity.Project {
	projectUUID := uuid.UUID(dbProject.ID.Bytes)

	return &entity.Project{
		ID:          projectUUID.String(),
		Title:       dbProject.Title,
		Description: dbProject.Description.String,
		CreatedAt:   dbProject.CreatedAt.Time,
	}
}

func toEntityFile(dbFile *sqlc.ProjectFile) *entity.File {
	fileUUID := uuid.UUID(dbFile.ID.Bytes)
	projectUUID := uuid.UUID(dbFile.ProjectID.Bytes)

	return &entity.File{
		ID:          fileUUID.String(),
		ProjectID:   projectUUID.String(),
		Filename:    dbFile.Filename,
		Size:        dbFile.Size,
		ContentType: dbFile.ContentType,
		CreatedAt:   dbFile.CreatedAt.Time,
	}
}

func toEntitySession(dbSession *sqlc.Session) *entity.Session {
	sessionUUID := uuid.UUID(dbSession.ID.Bytes)

	session := &entity.Session{
		ID:               sessionUUID.String(),
		Status:           entity.SessionStatus(dbSession.Status),
		CurrentIteration: int(dbSession.CurrentIteration),
		CreatedAt:        dbSession.CreatedAt.Time,
		UpdatedAt:        dbSession.UpdatedAt.Time,
	}

	if dbSession.ProjectID.Valid {
		projectUUID := uuid.UUID(dbSession.ProjectID.Bytes)
		projectIDStr := projectUUID.String()
		session.ProjectID = &projectIDStr
	}

	if dbSession.Type.Valid {
		sessionType := entity.SessionType(dbSession.Type.String)
		session.Type = &sessionType
	}

	if dbSession.UserGoal.Valid {
		userGoal := dbSession.UserGoal.String
		session.UserGoal = &userGoal
	}

	if dbSession.ProjectContext.Valid {
		projectContext := dbSession.ProjectContext.String
		session.ProjectContext = &projectContext
	}

	if dbSession.Result.Valid {
		result := dbSession.Result.String
		session.Result = &result
	}

	if dbSession.Error.Valid {
		errorMsg := dbSession.Error.String
		session.Error = &errorMsg
	}

	return session
}

func toEntityIteration(dbIter *sqlc.SessionIteration) *entity.Iteration {
	iterUUID := uuid.UUID(dbIter.ID.Bytes)
	sessionUUID := uuid.UUID(dbIter.SessionID.Bytes)

	iteration := &entity.Iteration{
		ID:              iterUUID.String(),
		SessionID:       sessionUUID.String(),
		Title:           dbIter.Title,
		IterationNumber: int(dbIter.IterationNumber),
		CreatedAt:       dbIter.CreatedAt.Time,
	}

	return iteration
}

func toEntityQuestion(dbQuestion *sqlc.IterationQuestion) *entity.Question {
	questionUUID := uuid.UUID(dbQuestion.ID.Bytes)
	iterationUUID := uuid.UUID(dbQuestion.IterationID.Bytes)

	question := &entity.Question{
		ID:             questionUUID.String(),
		IterationID:    iterationUUID.String(),
		QuestionNumber: int(dbQuestion.QuestionNumber),
		Status:         entity.QuestionStatus(dbQuestion.Status),
		Question:       dbQuestion.Question,
		Explanation:    dbQuestion.Explanation,
		CreatedAt:      dbQuestion.CreatedAt.Time,
	}

	if dbQuestion.Answer.Valid {
		answer := dbQuestion.Answer.String
		question.Answer = &answer
	}

	if dbQuestion.AnsweredAt.Valid {
		answeredAt := dbQuestion.AnsweredAt.Time
		question.AnsweredAt = &answeredAt
	}

	return question
}

func toEntitySessionMessage(dbMsg *sqlc.SessionMessage) *entity.SessionMessage {
	msgUUID := uuid.UUID(dbMsg.ID.Bytes)
	sessionUUID := uuid.UUID(dbMsg.SessionID.Bytes)

	return &entity.SessionMessage{
		ID:          msgUUID.String(),
		SessionID:   sessionUUID.String(),
		MessageText: dbMsg.MessageText,
		CreatedAt:   dbMsg.CreatedAt.Time,
	}
}
