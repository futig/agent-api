package entity

type RAGChunk struct {
	Text string `json:"text"`
}

type RAGRelevantContext struct {
	RelevantChunks []RAGChunk `json:"relevant_chunks"`
}

type RAGGetContextRequest struct {
	ProjectID    string `json:"project_id"`
	UserGoal     string `json:"user_goal"`
	TopK         int    `json:"top_k"`
	MaxQuestions int    `json:"max_questions"`
}

type RAGGetContextResponse struct {
	RelevantContext RAGRelevantContext `json:"relevant_context"`
}

type RAGDeleteIndexResponse struct {
	DeletedCount int `json:"deleted_count,omitempty"`
}

type FileData struct {
	Filename string
	Content  []byte
}
