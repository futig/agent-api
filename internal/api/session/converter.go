package session

import "github.com/futig/agent-backend/internal/entity"

// toSessionDTO converts Session entity to SessionDTO
func toSessionDTO(session *entity.Session) *entity.SessionDTO {
	return &entity.SessionDTO{
		ID:               session.ID,
		ProjectID:        session.ProjectID,
		Status:           session.Status,
		CurrentIteration: session.CurrentIteration,
		Result:           session.Result,
		Error:            session.Error,
		CreatedAt:        session.CreatedAt,
		UpdatedAt:        session.UpdatedAt,
	}
}
