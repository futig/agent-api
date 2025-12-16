package session

import (
	"context"
	"fmt"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/pkg/validator"
	"github.com/futig/agent-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// SessionUsecase implements session business logic
type SessionUsecase struct {
	sessionRepo        repository.SessionRepository
	iterationRepo      repository.IterationRepository
	questionRepo       repository.QuestionRepository
	projectRepo        repository.ProjectRepository
	sessionMessageRepo repository.SessionMessageRepository
	validator          *validator.Validator
	ragConnector       RagConnector
	llmConnector       LLMConnector
	asrConnector       ASRConnector
	logger             *zap.Logger
}

// NewUsecase creates a new session use case
func NewUsecase(
	sessionRepo repository.SessionRepository,
	iterationRepo repository.IterationRepository,
	questionRepo repository.QuestionRepository,
	projectRepo repository.ProjectRepository,
	sessionMessageRepo repository.SessionMessageRepository,
	validator *validator.Validator,
	ragConnector RagConnector,
	llmConnector LLMConnector,
	asrConnector ASRConnector,
	logger *zap.Logger,
) *SessionUsecase {
	return &SessionUsecase{
		sessionRepo:        sessionRepo,
		iterationRepo:      iterationRepo,
		questionRepo:       questionRepo,
		projectRepo:        projectRepo,
		sessionMessageRepo: sessionMessageRepo,
		validator:          validator,
		ragConnector:       ragConnector,
		llmConnector:       llmConnector,
		asrConnector:       asrConnector,
		logger:             logger,
	}
}

// StartSession creates an empty session in the database
func (uc *SessionUsecase) StartSession(ctx context.Context) (*entity.Session, error) {
	session := entity.Session{
		ID:     uuid.New().String(),
		Status: entity.SessionStatusAskUserGoal,
	}

	createdSession, err := uc.sessionRepo.CreateSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return createdSession, nil
}

// SubmitAudioUserGoal transcribes audio and submits the goal as text
func (uc *SessionUsecase) SubmitAudioUserGoal(ctx context.Context, sessionID string, audioGoal []byte) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusAskUserGoal {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	transcription, err := uc.transcribeAudio(ctx, sessionID, audioGoal)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}

	return uc.SubmitTextUserGoal(ctx, sessionID, transcription)
}

// SubmitTextUserGoal saves the user goal to the session
func (uc *SessionUsecase) SubmitTextUserGoal(ctx context.Context, sessionID, goal string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusAskUserGoal {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	_, err = uc.sessionRepo.UpdateSessionUserGoal(ctx, sessionID, goal)
	if err != nil {
		return nil, fmt.Errorf("update user goal: %w", err)
	}

	session, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusSelectOrCreateProject)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return session, nil
}

// SubmitRAGProjectContext generates RAG context for the project and saves it
func (uc *SessionUsecase) SubmitRAGProjectContext(ctx context.Context, sessionID, projectID string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusSelectOrCreateProject {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	if session.UserGoal == nil || *session.UserGoal == "" {
		return nil, fmt.Errorf("user goal must be set before generating context")
	}

	_, err = uc.projectRepo.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	ragContext, err := uc.ragConnector.GetContext(ctx, &entity.RAGGetContextRequest{
		ProjectID:    projectID,
		UserGoal:     *session.UserGoal,
		TopK:         5,
		MaxQuestions: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("get RAG context: %w", err)
	}

	_, err = uc.sessionRepo.UpdateSessionRAGProjectContext(ctx, sessionID, projectID, ragContext)
	if err != nil {
		return nil, fmt.Errorf("update project context: %w", err)
	}

	session, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusChooseMode)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return session, nil
}

// SubmitAudioUserProjectContext transcribes audio and submits manual context
func (uc *SessionUsecase) SubmitAudioUserProjectContext(ctx context.Context, sessionID, questions string, audioAnswers []byte) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusAskUserContext {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	transcription, err := uc.transcribeAudio(ctx, sessionID, audioAnswers)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}

	return uc.SubmitTextUserProjectContext(ctx, sessionID, questions, transcription)
}

// SubmitTextUserProjectContext formats and saves manual context from Q&A
func (uc *SessionUsecase) SubmitTextUserProjectContext(ctx context.Context, sessionID, questions, answers string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusAskUserContext {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	formattedContext := fmt.Sprintf("На вопросы %s пользователь ответил: %s", questions, answers)

	_, err = uc.sessionRepo.UpdateSessionProjectContext(ctx, sessionID, formattedContext)
	if err != nil {
		return nil, fmt.Errorf("update project context: %w", err)
	}

	session, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusChooseMode)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return session, nil
}

// SetSessionType sets the session type (Interview or Draft mode)
func (uc *SessionUsecase) SetSessionType(ctx context.Context, sessionID string, sessionType entity.SessionType) (*entity.Session, error) {
	if err := sessionType.Validate(); err != nil {
		return nil, err
	}

	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusChooseMode {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	_, err = uc.sessionRepo.UpdateSessionType(ctx, sessionID, sessionType)
	if err != nil {
		return nil, fmt.Errorf("update session type: %w", err)
	}

	var status entity.SessionStatus
	switch sessionType {
	case entity.SessionTypeInterview:
		status = entity.SessionStatusInterviewInfo
	case entity.SessionTypeDraft:
		status = entity.SessionStatusDraftInfo
	default:
	}

	session, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, status)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return session, nil
}

// StartManualContext switches session from SELECT_OR_CREATE_PROJECT to ASK_USER_CONTEXT
func (uc *SessionUsecase) StartManualContext(ctx context.Context, sessionID string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusSelectOrCreateProject {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	updated, err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusAskUserContext)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return updated, nil
}

// RestartModeSelection switches session from INTERVIEW_INFO/DRAFT_INFO back to CHOOSE_MODE
// so that user can change the mode selection.
func (uc *SessionUsecase) RestartModeSelection(ctx context.Context, sessionID string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusInterviewInfo && session.Status != entity.SessionStatusDraftInfo {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	updated, err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusChooseMode)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return updated, nil
}

// RestartProjectSelection switches session from CHOOSE_MODE back to SELECT_OR_CREATE_PROJECT
// so that user can re-select project or choose manual context again.
func (uc *SessionUsecase) RestartProjectSelection(ctx context.Context, sessionID string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusChooseMode {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	updated, err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusSelectOrCreateProject)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return updated, nil
}

// StartDraftCollecting switches draft session from DRAFT_INFO to DRAFT_COLLECTING
func (uc *SessionUsecase) StartDraftCollecting(ctx context.Context, sessionID string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Type == nil || *session.Type != entity.SessionTypeDraft {
		return nil, fmt.Errorf("wrong session type '%v' for draft collecting", session.Type)
	}

	if session.Status != entity.SessionStatusDraftInfo {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	updated, err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusDraftCollecting)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return updated, nil
}

// LoadSessionQuestions generates questions and saves them to the database
func (uc *SessionUsecase) LoadSessionQuestions(ctx context.Context, sessionID string) ([]*entity.IterationWithQuestions, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusInterviewInfo {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	if session.UserGoal == nil || *session.UserGoal == "" {
		return nil, fmt.Errorf("user goal must be set before generating questions")
	}

	if session.ProjectContext == nil || *session.ProjectContext == "" {
		return nil, fmt.Errorf("project context must be set before generating questions")
	}

	var projectDescription *string
	if session.ProjectID != nil && *session.ProjectID != "" {
		project, err := uc.projectRepo.Get(ctx, *session.ProjectID)
		if err != nil || project.Description == "" {
			return nil, fmt.Errorf("get project description: %w", err)
		}
		projectDescription = &project.Description
	}

	blocks, err := uc.generateQuestionsBlocks(ctx, *session.UserGoal, *session.ProjectContext, projectDescription)
	if err != nil {
		return nil, fmt.Errorf("generate questions: %w", err)
	}

	savedIterations, err := uc.saveQuestionsToDatabase(ctx, sessionID, blocks)
	if err != nil {
		return nil, fmt.Errorf("save questions: %w", err)
	}

	// Update session status to waiting for answers
	_, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusWaitingForAnswers)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	_, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusWaitingForAnswers)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	ctxzap.Info(ctx, "questions loaded successfully",
		zap.String("session_id", sessionID),
		zap.Int("iteration_count", len(blocks)),
	)

	return savedIterations, nil
}

// SkipAnswer marks a question as skipped and returns the next question block
func (uc *SessionUsecase) SkipAnswer(ctx context.Context, sessionID, questionID string) (*entity.IterationWithQuestions, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusWaitingForAnswers {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	if err := uc.questionRepo.SkipQuestion(ctx, questionID); err != nil {
		return nil, fmt.Errorf("skip question: %w", err)
	}

	iteration, err := uc.getCurrentIteration(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get current/next iteration: %w", err)
	}

	if iteration == nil {
		_, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusValidating)
		if err != nil {
			return nil, fmt.Errorf("update session status: %w", err)
		}
	}

	return iteration, nil
}

func (uc *SessionUsecase) SubmitAudioAnswer(ctx context.Context, sessionID, questionID string, audioAnswer []byte) (*entity.IterationWithQuestions, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusWaitingForAnswers {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	transcription, err := uc.transcribeAudio(ctx, sessionID, audioAnswer)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}

	return uc.SubmitTextAnswer(ctx, sessionID, questionID, transcription)
}

func (uc *SessionUsecase) SubmitTextAnswer(ctx context.Context, sessionID, questionID, answer string) (*entity.IterationWithQuestions, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusWaitingForAnswers {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	if err := uc.questionRepo.UpdateQuestionAnswer(ctx, questionID, answer); err != nil {
		return nil, fmt.Errorf("save answer: %w", err)
	}

	iteration, err := uc.getCurrentIteration(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get current/next iteration: %w", err)
	}

	// Only change to VALIDATING if there are truly no more questions to answer
	// (including skipped/unanswered questions)
	if iteration == nil {
		// Check if there are any unanswered questions (skipped questions)
		unansweredQuestions, err := uc.questionRepo.GetUnansweredQuestions(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("check unanswered questions: %w", err)
		}

		// Only move to VALIDATING if there are no unanswered questions at all
		if len(unansweredQuestions) == 0 {
			_, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusValidating)
			if err != nil {
				return nil, fmt.Errorf("update session status: %w", err)
			}
		}
	}

	return iteration, nil
}

// GetUnansweredQuestions returns all unanswered and skipped questions for a session
func (uc *SessionUsecase) GetUnansweredQuestions(ctx context.Context, sessionID string) ([]*entity.Question, error) {
	questions, err := uc.questionRepo.GetUnansweredQuestions(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get unanswered questions: %w", err)
	}

	return questions, nil
}

// SkipAnswer marks a question as skipped and returns the next question block
func (uc *SessionUsecase) SkipSkipedQuestion(ctx context.Context, sessionID, questionID string) ([]*entity.Question, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusWaitingForAnswers {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	if err := uc.questionRepo.SkipQuestion(ctx, questionID); err != nil {
		return nil, fmt.Errorf("skip question: %w", err)
	}

	questions, err := uc.questionRepo.GetUnansweredQuestions(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get unanswered questions: %w", err)
	}

	if len(questions) == 0 {
		_, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusValidating)
		if err != nil {
			return nil, fmt.Errorf("update session status: %w", err)
		}
	}

	return questions, nil
}

// RestartValidationFromDone moves session from DONE back into validation flow.
// It is used when user chooses to answer or skip remaining questions after
// the initial summary has already been generated.
func (uc *SessionUsecase) SetWaitingForAnswersStatus(ctx context.Context, sessionID string) error {
	_, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if _, err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusWaitingForAnswers); err != nil {
		return fmt.Errorf("update session status to waiting for answers: %w", err)
	}

	return nil
}

// GetQuestionExplanation returns explanation text for a given question
func (uc *SessionUsecase) GetQuestionExplanation(ctx context.Context, questionID string) (string, error) {
	question, err := uc.questionRepo.GetQuestionByID(ctx, questionID)
	if err != nil {
		return "", fmt.Errorf("get question: %w", err)
	}

	return question.Explanation, nil
}

// GetQuestionByID returns a question by ID
func (uc *SessionUsecase) GetQuestionByID(ctx context.Context, questionID string) (*entity.Question, error) {
	question, err := uc.questionRepo.GetQuestionByID(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("get question: %w", err)
	}

	return question, nil
}

// GetIterationByID returns an iteration with all its questions
func (uc *SessionUsecase) GetIterationByID(ctx context.Context, iterationID string) (*entity.IterationWithQuestions, error) {
	iteration, err := uc.iterationRepo.GetIterationByID(ctx, iterationID)
	if err != nil {
		return nil, fmt.Errorf("get iteration: %w", err)
	}

	questions, err := uc.questionRepo.ListQuestionsByIteration(ctx, iterationID)
	if err != nil {
		return nil, fmt.Errorf("get questions: %w", err)
	}

	// Convert to DTOs
	questionDTOs := make([]entity.QuestionDTO, 0, len(questions))
	for _, q := range questions {
		questionDTOs = append(questionDTOs, entity.QuestionDTO{
			ID:             q.ID,
			QuestionNumber: q.QuestionNumber,
			Question:       q.Question,
			Explanation:    q.Explanation,
			Status:         q.Status,
		})
	}

	return &entity.IterationWithQuestions{
		IterationNumber: iteration.IterationNumber,
		SessionID: iteration.SessionID,
		IterationID: iteration.ID,
		Title:       iteration.Title,
		Questions:   questionDTOs,
	}, nil
}

// ValidateAnswers validates completeness of answers and may return additional questions
func (uc *SessionUsecase) ValidateAnswers(ctx context.Context, sessionID string) (*entity.IterationWithQuestions, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusValidating && session.Status != entity.SessionStatusWaitingForAnswers {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	if session.UserGoal == nil || *session.UserGoal == "" {
		return nil, fmt.Errorf("user goal not set")
	}

	if session.ProjectContext == nil || *session.ProjectContext == "" {
		return nil, fmt.Errorf("project context not set")
	}

	iterations, err := uc.iterationRepo.ListIterationsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list iterations before additional questions: %w", err)
	}

	hasAdditionalBlock := false
	for _, it := range iterations {
		if it.Title == "Дополнительные вопросы" {
			hasAdditionalBlock = true
			break
		}
	}

	if hasAdditionalBlock {
		ctxzap.Info(ctx, "additional questions block already exists, skipping extra generation",
			zap.String("session_id", sessionID),
			zap.Int("current_iteration", session.CurrentIteration),
		)

		// Сразу переходим к генерации требований
		_, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusGeneratingRequirements)
		if err != nil {
			return nil, fmt.Errorf("update session status: %w", err)
		}

		return nil, nil
	}

	allAnswers, err := uc.collectAllAnswers(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("collect answers: %w", err)
	}

	validateReq := &entity.LLMValidateAnswersRequest{
		UserGoal:          *session.UserGoal,
		ProjectContext:    *session.ProjectContext,
		CompleteQuestions: allAnswers,
	}

	validateResp, err := uc.llmConnector.ValidateAnswers(ctx, validateReq)
	if err != nil {
		return nil, fmt.Errorf("validate answers: %w", err)
	}

	status := entity.SessionStatusGeneratingRequirements
	var additionalIteration *entity.IterationWithQuestions

	if len(validateResp.Questions) != 0 {

		savedIterations, err := uc.saveQuestionsToDatabase(ctx, sessionID, []entity.QuestionsBlock{
			{
				Title:     "Дополнительные вопросы",
				Questions: validateResp.Questions,
			},
		})
		if err != nil || len(savedIterations) == 0 {
			return nil, fmt.Errorf("save questions: %w", err)
		}

		additionalIteration = savedIterations[0]
		status = entity.SessionStatusWaitingForAnswers
	}

	_, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, status)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return additionalIteration, nil
}

// GenerateSummaty generates final requirements from all answers
func (uc *SessionUsecase) GenerateSummary(ctx context.Context, sessionID string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusGeneratingRequirements && session.Status != entity.SessionStatusWaitingForAnswers {
		return nil, fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	if session.UserGoal == nil || *session.UserGoal == "" {
		return nil, fmt.Errorf("user goal not set")
	}

	if session.ProjectContext == nil || *session.ProjectContext == "" {
		return nil, fmt.Errorf("project context not set")
	}

	allAnswers, err := uc.collectAllAnswers(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("collect answers: %w", err)
	}

	summaryReq := &entity.LLMGenerateSummaryRequest{
		UserGoal:          *session.UserGoal,
		ProjectContext:    *session.ProjectContext,
		CompleteQuestions: allAnswers,
	}

	summaryResp, err := uc.llmConnector.GenerateSummary(ctx, summaryReq)
	if err != nil {
		return nil, fmt.Errorf("generate summary: %w", err)
	}

	updatedSession, err := uc.sessionRepo.UpdateSessionResult(ctx, sessionID, entity.SessionStatusDone, &summaryResp, nil)
	if err != nil {
		return nil, fmt.Errorf("save summary: %w", err)
	}

	return updatedSession, nil
}

// GetSession retrieves a session by ID
func (uc *SessionUsecase) GetSession(ctx context.Context, sessionID string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return session, nil
}

func (uc *SessionUsecase) GetSessionResult(ctx context.Context, sessionID string) (string, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusDone {
		return "", fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	if session.Result == nil || *session.Result == "" {
		return "", entity.ErrNoResult
	}

	return *session.Result, nil
}

// CancelSession cancels an active session
func (uc *SessionUsecase) CancelSession(ctx context.Context, sessionID string) error {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if session.Status == entity.SessionStatusDone || session.Status == entity.SessionStatusCanceled {
		return fmt.Errorf("wrong action on status '%s'", session.Status)
	}

	if _, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusCanceled); err != nil {
		return fmt.Errorf("cancel session: %w", err)
	}

	return nil
}

// UpdateSessionStatus updates the session status
func (uc *SessionUsecase) UpdateSessionStatus(ctx context.Context, sessionID string, status entity.SessionStatus) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	updatedSession, err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, status)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	ctxzap.Info(ctx, "session status updated",
		zap.String("session_id", sessionID),
		zap.String("old_status", string(session.Status)),
		zap.String("new_status", string(status)),
	)

	return updatedSession, nil
}

// AddDraftMessage adds a text draft message to a session
func (uc *SessionUsecase) AddDraftMessage(
	ctx context.Context,
	sessionID,
	messageText string,
) (*entity.SessionMessage, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusDraftCollecting {
		return nil, fmt.Errorf("invalid session status for adding draft message: %s", session.Status)
	}

	msg, err := uc.sessionMessageRepo.CreateMessage(ctx, sessionID, messageText)
	if err != nil {
		return nil, fmt.Errorf("create draft message: %w", err)
	}

	return msg, nil
}

// AddAudioDraftMessage transcribes audio and adds it as a draft message
func (uc *SessionUsecase) AddAudioDraftMessage(
	ctx context.Context,
	sessionID string,
	audioData []byte,
) (*entity.SessionMessage, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusDraftCollecting {
		return nil, fmt.Errorf("invalid session status for adding draft message: %s", session.Status)
	}

	transcription, err := uc.transcribeAudio(ctx, sessionID, audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}

	return uc.AddDraftMessage(ctx, sessionID, transcription)
}

// ValidateDraftMessages validates collected draft messages and may return additional questions
func (uc *SessionUsecase) ValidateDraftMessages(
	ctx context.Context,
	sessionID string,
) (*entity.IterationWithQuestions, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusDraftCollecting && session.Status != entity.SessionStatusWaitingForAnswers && session.Status != entity.SessionStatusValidating {
		return nil, fmt.Errorf("invalid session status for validation: %s", session.Status)
	}

	if session.UserGoal == nil || *session.UserGoal == "" {
		return nil, fmt.Errorf("user goal not set")
	}

	if session.ProjectContext == nil || *session.ProjectContext == "" {
		return nil, fmt.Errorf("project context not set")
	}

	iterations, err := uc.iterationRepo.ListIterationsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list iterations before additional questions: %w", err)
	}

	hasAdditionalBlock := false
	for _, it := range iterations {
		if it.Title == "Дополнительные вопросы" {
			hasAdditionalBlock = true
			break
		}
	}

	if hasAdditionalBlock {
		ctxzap.Info(ctx, "additional questions block already exists, skipping extra generation",
			zap.String("session_id", sessionID),
			zap.Int("current_iteration", session.CurrentIteration),
		)

		// Сразу переходим к генерации требований
		_, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusGeneratingRequirements)
		if err != nil {
			return nil, fmt.Errorf("update session status: %w", err)
		}

		return nil, nil
	}

	if _, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusValidating); err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	messages, err := uc.sessionMessageRepo.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session messages: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no draft messages to validate")
	}

	questions, err := uc.questionRepo.ListQuestionsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get questions by session: %w", err)
	}

	additionalQuestions := make([]entity.QuestionWithAnswer, 0, len(questions))
	for _, q := range questions {
		if q.Answer != nil {
			additionalQuestions = append(additionalQuestions, entity.QuestionWithAnswer{
				Question: q.Question,
				Answer:   *q.Answer,
			})
		}
	}

	messageTexts := make([]string, 0, len(messages))
	for _, m := range messages {
		messageTexts = append(messageTexts, m.MessageText)
	}

	var projectDescription *string
	if session.ProjectID != nil && *session.ProjectID != "" {
		project, err := uc.projectRepo.Get(ctx, *session.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("get project description: %w", err)
		}
		projectDescription = &project.Description
	}

	req := &entity.LLMValidateDraftRequest{
		Messages:            messageTexts,
		AdditionalQuestions: additionalQuestions,
		UserGoal:            *session.UserGoal,
		ProjectContext:      *session.ProjectContext,
		ProjectDescription:  projectDescription,
	}

	validateResp, err := uc.llmConnector.ValidateDraft(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("validate draft: %w", err)
	}

	if len(validateResp.Questions) == 0 {
		if _, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusGeneratingRequirements); err != nil {
			return nil, fmt.Errorf("update session status: %w", err)
		}
		return nil, nil
	}

	blocks := []entity.QuestionsBlock{
		{
			Title:     "Дополнительные вопросы",
			Questions: validateResp.Questions,
		},
	}

	savedIterations, err := uc.saveQuestionsToDatabase(ctx, sessionID, blocks)
	if err != nil || len(savedIterations) == 0 {
		return nil, fmt.Errorf("save questions: %w", err)
	}

	if _, err = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, entity.SessionStatusWaitingForAnswers); err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return savedIterations[0], nil
}

// GenerateDraftSummary generates final business requirements from draft messages and answers
func (uc *SessionUsecase) GenerateDraftSummary(ctx context.Context, sessionID string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Status != entity.SessionStatusGeneratingRequirements && session.Status != entity.SessionStatusWaitingForAnswers {
		return nil, fmt.Errorf("invalid session status: %s", session.Status)
	}

	if session.UserGoal == nil || *session.UserGoal == "" {
		return nil, fmt.Errorf("user goal not set")
	}

	if session.ProjectContext == nil || *session.ProjectContext == "" {
		return nil, fmt.Errorf("project context not set")
	}

	messages, err := uc.sessionMessageRepo.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session messages: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no draft messages to generate summary")
	}

	additionalQuestions, err := uc.collectAllAnswers(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("collect answers: %w", err)
	}

	messageTexts := make([]string, 0, len(messages))
	for _, m := range messages {
		messageTexts = append(messageTexts, m.MessageText)
	}

	var projectDescription *string
	if session.ProjectID != nil && *session.ProjectID != "" {
		project, err := uc.projectRepo.Get(ctx, *session.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("get project description: %w", err)
		}
		projectDescription = &project.Description
	}

	req := &entity.LLMGenerateDraftSummaryRequest{
		Messages:            messageTexts,
		AdditionalQuestions: additionalQuestions,
		UserGoal:            *session.UserGoal,
		ProjectContext:      *session.ProjectContext,
		ProjectDescription:  projectDescription,
	}

	summary, err := uc.llmConnector.GenerateDraftSummary(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generate draft summary: %w", err)
	}

	updatedSession, err := uc.sessionRepo.UpdateSessionResult(
		ctx,
		sessionID,
		entity.SessionStatusDone,
		&summary,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("save draft summary: %w", err)
	}

	return updatedSession, nil
}
