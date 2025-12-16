package entity

type UserContext struct {
	Goal  string `json:"goal"`
	Task  string `json:"task"`
	Value string `json:"value"`
}

type LLMGenerateQuestionsRequest struct {
	UserGoal           string  `json:"user_goal"`
	ProjectContext     string  `json:"project_context"`
	ProjectDescription *string `json:"project_description,omitempty"`
}

type LLMQuestion struct {
	Text        string `json:"text"`
	Explanation string `json:"explanation"`
}

type QuestionsBlock struct {
	Title     string        `json:"title"`
	Questions []LLMQuestion `json:"questions"`
}

type LLMGenerateQuestionsResponse struct {
	Iterations []QuestionsBlock `json:"iterations"`
}

type QuestionWithAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type LLMValidateAnswersRequest struct {
	CompleteQuestions  []QuestionWithAnswer `json:"answered_questions"`
	UserGoal           string               `json:"user_goal"`
	ProjectContext     string               `json:"project_context"`
	ProjectDescription *string              `json:"project_description,omitempty"`
}

type LLMValidateAnswersResponse struct {
	Questions []LLMQuestion `json:"questions"`
}

type LLMGenerateSummaryRequest struct {
	CompleteQuestions  []QuestionWithAnswer `json:"answered_questions"`
	UserGoal           string               `json:"user_goal"`
	ProjectContext     string               `json:"project_context"`
	ProjectDescription *string              `json:"project_description,omitempty"`
}

type LLMGenerateSummaryResponse struct {
	Result string `json:"result"`
}

type LLMValidateDraftRequest struct {
	Messages            []string             `json:"messages"`
	AdditionalQuestions []QuestionWithAnswer `json:"additional_questions"`
	UserGoal            string               `json:"user_goal"`
	ProjectContext      string               `json:"project_context"`
	ProjectDescription  *string              `json:"project_description,omitempty"`
}

type LLMGenerateDraftSummaryRequest struct {
	Messages            []string             `json:"messages"`
	AdditionalQuestions []QuestionWithAnswer `json:"additional_questions"`
	UserGoal            string               `json:"user_goal"`
	ProjectContext      string               `json:"project_context"`
	ProjectDescription  *string              `json:"project_description,omitempty"`
}
