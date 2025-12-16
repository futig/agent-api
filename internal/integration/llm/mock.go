package llm

import (
	"context"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// MockConnector - мок-реализация LLM коннектора для тестирования
type MockConnector struct {
	logger *zap.Logger
}

func NewMockConnector(logger *zap.Logger) *MockConnector {
	return &MockConnector{
		logger: logger,
	}
}

// GenerateQuestions - мок генерации вопросов
func (m *MockConnector) GenerateQuestions(ctx context.Context, req *entity.LLMGenerateQuestionsRequest) (
	*entity.LLMGenerateQuestionsResponse, error,
) {
	ctxzap.Info(ctx, "[MOCK] generating questions via LLM")

	// Возвращаем 5 итераций с вопросами
	resp := &entity.LLMGenerateQuestionsResponse{
		Iterations: []entity.QuestionsBlock{
			{
				Title: "Общая информация о проекте",
				Questions: []entity.LLMQuestion{
					{
						Text:        "Опишите основную цель и назначение системы?",
						Explanation: "Необходимо понять общее видение проекта",
					},
					{
						Text:        "Кто является целевой аудиторией данного проекта?",
						Explanation: "Важно определить основных пользователей системы",
					},
					{
						Text:        "Какие бизнес-задачи должна решать система?",
						Explanation: "Понимание бизнес-ценности проекта",
					},
				},
			},
			{
				Title: "Функциональные требования",
				Questions: []entity.LLMQuestion{
					{
						Text:        "Какие основные функции должна предоставлять система?",
						Explanation: "Определение ключевой функциональности",
					},
					{
						Text:        "Опишите основные пользовательские сценарии работы с системой?",
						Explanation: "Понимание user flow и взаимодействия с системой",
					},
					{
						Text:        "Какие данные будет обрабатывать система?",
						Explanation: "Определение типов данных и их структуры",
					},
				},
			},
			{
				Title: "Нефункциональные требования",
				Questions: []entity.LLMQuestion{
					{
						Text:        "Какие требования к производительности системы?",
						Explanation: "Определение ожидаемой нагрузки и скорости работы",
					},
					{
						Text:        "Какие требования к безопасности данных?",
						Explanation: "Понимание уровня защиты информации",
					},
					{
						Text:        "Какова ожидаемая доступность системы (uptime)?",
						Explanation: "Определение требований к надежности",
					},
				},
			},
			{
				Title: "Интеграции и внешние системы",
				Questions: []entity.LLMQuestion{
					{
						Text:        "С какими внешними системами должна интегрироваться система?",
						Explanation: "Определение внешних зависимостей",
					},
					{
						Text:        "Какие API или протоколы должны использоваться для интеграции?",
						Explanation: "Технические детали интеграции",
					},
					{
						Text:        "Есть ли требования к формату обмена данными?",
						Explanation: "Определение форматов данных (JSON, XML и т.д.)",
					},
				},
			},
			{
				Title: "Ограничения и предположения",
				Questions: []entity.LLMQuestion{
					{
						Text:        "Какие технические ограничения существуют для проекта?",
						Explanation: "Понимание технических рамок проекта",
					},
					{
						Text:        "Какие временные и бюджетные ограничения проекта?",
						Explanation: "Определение ресурсных рамок",
					},
					{
						Text:        "Какие риски и предположения существуют для данного проекта?",
						Explanation: "Выявление потенциальных проблем",
					},
				},
			},
		},
	}

	ctxzap.Info(ctx, "[MOCK] questions generated", zap.Int("block_count", len(resp.Iterations)))
	return resp, nil
}

// ValidateAnswers - мок валидации ответов
func (m *MockConnector) ValidateAnswers(ctx context.Context, req *entity.LLMValidateAnswersRequest) (
	*entity.LLMValidateAnswersResponse, error,
) {
	ctxzap.Info(ctx, "[MOCK] validating answers via LLM")

	// Мок всегда возвращает пустой список вопросов (ответы достаточны)
	resp := &entity.LLMValidateAnswersResponse{
		Questions: []entity.LLMQuestion{},
	}

	ctxzap.Info(ctx, "[MOCK] answers validated", zap.Int("additional_questions", len(resp.Questions)))
	return resp, nil
}

// GenerateSummary - мок генерации итогового резюме
func (m *MockConnector) GenerateSummary(ctx context.Context, req *entity.LLMGenerateSummaryRequest) (string, error) {
	ctxzap.Info(ctx, "[MOCK] generating summary via LLM")

	summary := `# Бизнес-требования (MOCK)

## 1. Обзор проекта
Данный документ содержит бизнес-требования, собранные на основе интервью с заинтересованными сторонами.

## 2. Функциональные требования

### 2.1 Основная функциональность
- Система должна предоставлять базовую функциональность для целевой аудитории
- Необходимо обеспечить простой и интуитивный интерфейс пользователя

### 2.2 Аутентификация и авторизация
- Пользователи должны иметь возможность регистрироваться в системе
- Система должна поддерживать различные роли пользователей

## 3. Нефункциональные требования

### 3.1 Производительность
- Время отклика системы не должно превышать 2 секунд
- Система должна поддерживать до 1000 одновременных пользователей

### 3.2 Безопасность
- Все данные должны передаваться по защищенному соединению (HTTPS)
- Пароли пользователей должны храниться в зашифрованном виде

## 4. Ограничения и предположения
- Проект должен быть реализован в течение 6 месяцев
- Бюджет проекта ограничен

---
*Документ сгенерирован автоматически (MOCK)*`

	ctxzap.Info(ctx, "[MOCK] summary generated", zap.Int("result_length", len(summary)))
	return summary, nil
}

// ValidateDraft - мок валидации черновика
func (m *MockConnector) ValidateDraft(ctx context.Context, req *entity.LLMValidateDraftRequest) (
	*entity.LLMValidateAnswersResponse, error,
) {
	ctxzap.Info(ctx, "[MOCK] validating draft via LLM")

	// Мок всегда возвращает пустой список вопросов (черновик готов)
	resp := &entity.LLMValidateAnswersResponse{
		Questions: []entity.LLMQuestion{},
	}

	ctxzap.Info(ctx, "[MOCK] draft validated", zap.Int("additional_questions", len(resp.Questions)))
	return resp, nil
}

// GenerateDraftSummary - мок генерации резюме черновика
func (m *MockConnector) GenerateDraftSummary(ctx context.Context, req *entity.LLMGenerateDraftSummaryRequest) (string, error) {
	ctxzap.Info(ctx, "[MOCK] generating draft summary via LLM")

	summary := `# Черновик бизнес-требований (MOCK)

## Основные выводы
На основе предоставленной информации были сформулированы следующие требования:

1. Требуется реализация базовой функциональности
2. Необходима поддержка аутентификации пользователей
3. Система должна быть масштабируемой

---
*Черновик сгенерирован автоматически (MOCK)*`

	ctxzap.Info(ctx, "[MOCK] draft summary generated", zap.Int("result_length", len(summary)))
	return summary, nil
}
