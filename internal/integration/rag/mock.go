package rag

import (
	"context"
	"fmt"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// MockConnector - мок-реализация RAG коннектора для тестирования
type MockConnector struct {
	logger *zap.Logger
}

func NewMockConnector(logger *zap.Logger) *MockConnector {
	return &MockConnector{
		logger: logger,
	}
}

// IndexFiles - мок индексации файлов
func (m *MockConnector) IndexFiles(ctx context.Context, projectID string, files []entity.FileData) error {
	ctxzap.Info(ctx, "[MOCK] indexing files in RAG",
		zap.String("project_id", projectID),
		zap.Int("file_count", len(files)),
	)
	return nil
}

// DeleteIndex - мок удаления индекса
func (m *MockConnector) DeleteIndex(ctx context.Context, projectID string) error {
	ctxzap.Info(ctx, "[MOCK] deleting RAG index",
		zap.String("project_id", projectID),
	)
	return nil
}

// GetContext - мок получения контекста из RAG
func (m *MockConnector) GetContext(ctx context.Context, req *entity.RAGGetContextRequest) (string, error) {
	ctxzap.Info(ctx, "[MOCK] getting context from RAG",
		zap.String("project_id", req.ProjectID),
		zap.String("user_goal", req.UserGoal),
	)

	// Возвращаем сокращенный мок-ответ
	mockContext := fmt.Sprintf(`Проект: %s

Контекст проекта (мок данные):
- Это тестовый проект для разработки веб-приложения
- Требуется реализовать аутентификацию пользователей
- Необходима интеграция с внешним API
- Проект использует микросервисную архитектуру

Цель пользователя: %s`, req.ProjectID, req.UserGoal)

	ctxzap.Debug(ctx, "[MOCK] context retrieved",
		zap.Int("context_length", len(mockContext)),
	)

	return mockContext, nil
}
