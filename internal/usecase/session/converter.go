package session

import "github.com/futig/agent-backend/internal/entity"

// questionsToIterationDTO converts an iteration and its questions to IterationWithQuestion DTO
func questionsToIterationDTO(iteration *entity.Iteration, questions []*entity.Question) *entity.IterationWithQuestions {
	if iteration == nil {
		return nil
	}

	questionDTOs := make([]entity.QuestionDTO, 0, len(questions))
	for _, q := range questions {
		if dto := questionModelToQuestionDTO(q); dto != nil {
			questionDTOs = append(questionDTOs, *dto)
		}
	}

	return &entity.IterationWithQuestions{
		SessionID:       iteration.SessionID,
		IterationID:     iteration.ID,
		IterationNumber: iteration.IterationNumber,
		Title:           iteration.Title,
		Questions:       questionDTOs,
	}
}

// questionModelToQuestionDTO converts a Question model to QuestionDTO
func questionModelToQuestionDTO(question *entity.Question) *entity.QuestionDTO {
	if question == nil {
		return nil
	}

	return &entity.QuestionDTO{
		ID:             question.ID,
		QuestionNumber: question.QuestionNumber,
		Status:         question.Status,
		Question:       question.Question,
		Explanation:    question.Explanation,
	}
}
