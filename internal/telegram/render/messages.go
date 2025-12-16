package render

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
)

const (
	// Welcome messages
	MsgWelcome = `👋 Привет! Я помогу превратить хаос мыслей в чёткие бизнес-требования.

Я умею:
• Провести структурированное интервью
• Собрать материалы в свободной форме
• Сгенерировать бизнес-требования`

	// User goal
	MsgAskGoal = `📋 Давай настроим контекст.

О чём проект? Можешь написать текстом или записать голосовое сообщение.`

	// Project selection
	MsgSelectProject = `📁 Отлично! Теперь выбери проект, в рамках которого будут вноситься изменения.

Или нажми "Проекта нет", если работаешь над новым проектом.`

	// Context questions
	MsgContextQuestion = `❓ %s

Ответь текстом или голосовым сообщением.`

	// Mode selection
	MsgChooseMode = `✅ Понял. В каком формате будет удобно продолжить работу?

📝 Интервью — я задам структурированные вопросы
📄 Драфт — пришли все материалы разом`

	// Interview info
	MsgInterviewInfo = `📝 Формат интервью

Тебе предстоит ответить на несколько вопросов, разделенных на блоки, по 3–4 в каждом.

⏱ Ориентировочно это займёт не больше 10 минут.

⚠️ Вопросы можно пропускать, но тогда бизнес-требования получатся не совсем полными.

Подходит такой вариант?`

	// Draft info
	MsgDraftInfo = `📄 Формат драфта

Пришли мне всё, что есть:
• Аудиозапись встречи (файл WAV)
• Пересланные сообщения из переписки
• Описание своими словами

📊 Можешь отправить до %d сообщений.

Готов начать?`

	// Question display
	MsgQuestion = `📌 %s

❓ Вопрос %d из %d: %s`

	// Draft collecting
	MsgDraftCollecting = `📥 Сообщение %d из %d принято.

Продолжай присылать материалы или нажми "Сформировать требования" когда будешь готов.`

	// Processing
	MsgProcessing = `⏳ Обрабатываю материалы и формирую бизнес-требования...

Это может занять несколько минут.`

	// Validation
	MsgValidating = `🔍 Проверяю полноту информации...`

	// Additional questions
	MsgAdditionalQuestions = `📋 Я изучил материалы. Мне не хватает информации по следующим пунктам:

%s

Ответь на дополнительные вопросы, чтобы требования были полными.`

	// Result ready
	MsgResultReady = `✅ Готово! Бизнес-требования сформированы.

Можешь скачать их в удобном формате:`

	// Session finished
	MsgSessionFinished = `👋 Сессия завершена.

Чтобы начать новую, нажми /start`

	// Errors
	ErrGeneric            = `❌ Произошла ошибка. Попробуйте ещё раз или нажмите /start`
	ErrTranscription      = `❌ Не удалось распознать голосовое сообщение. Попробуйте ещё раз или напишите текстом.`
	ErrSessionNotFound    = `❌ Сессия не найдена. Начните новую с /start`
	ErrInvalidState       = `❌ Неверное состояние. Нажмите /start чтобы начать заново.`
	ErrInvalidFile        = `❌ Неверный формат файла. Поддерживаются только WAV файлы.`
	ErrProjectNotFound    = `❌ Проект не найден. Попробуйте выбрать другой или создайте новый.`
	ErrMaxDraftMessages   = `❌ Достигнуто максимальное количество сообщений (%d). Нажмите "Сформировать требования".`
	ErrNetworkIssue       = `❌ Проблема с соединением. Попробуй чуть позже.`
	ErrServiceUnavailable = `❌ Сервис временно недоступен. Попробуй через пару минут.`
	ErrInvalidInput       = `❌ Неверный формат ответа. Попробуй по-другому.`
	ErrTimeout            = `❌ Операция заняла слишком много времени. Попробуй ещё раз.`
	ErrQuotaExceeded      = `❌ Превышен лимит запросов. Подожди немного.`
)

const (
	// MsgQuestionNoTitle is used for questions without iteration title
	MsgQuestionNoTitle = `❓ Вопрос %d из %d: %s`

	// MsgSkippedQuestion is used for skipped/unanswered questions after summary
	MsgSkippedQuestion = `❓ Пропущенный вопрос %d из %d: %s`
)

// RenderQuestion formats a question with context
func RenderQuestion(iterationTitle string, questionNumber, totalQuestions int, question string) string {
	if iterationTitle == "" {
		return fmt.Sprintf(MsgQuestionNoTitle, questionNumber, totalQuestions, question)
	}

	return fmt.Sprintf(MsgQuestion, iterationTitle, questionNumber, totalQuestions, question)
}

// RenderSkippedQuestion formats a question in the "answer skipped" flow
func RenderSkippedQuestion(currentNumber, totalQuestions int, question string) string {
	return fmt.Sprintf(MsgSkippedQuestion, currentNumber, totalQuestions, question)
}

// RenderAdditionalQuestions formats additional questions list
func RenderAdditionalQuestions(questions []string) string {
	var sb strings.Builder
	for i, q := range questions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, q))
	}
	return fmt.Sprintf(MsgAdditionalQuestions, sb.String())
}

// RenderInterviewInfo returns generic interview info text
func RenderInterviewInfo(questionCount, blockCount, estimatedMinutes int) string {
	return MsgInterviewInfo
}

// RenderDraftInfo formats draft info with message limit
func RenderDraftInfo(maxMessages int) string {
	return fmt.Sprintf(MsgDraftInfo, maxMessages)
}

// RenderDraftProgress formats draft collection progress with visual progress bar
func RenderDraftProgress(current, max int) string {
	progressBar := renderProgressBar(current, max)
	emoji := getProgressEmoji(current, max)

	return fmt.Sprintf("%s Сообщение %d из %d принято\n\n%s\n\nПродолжай присылать материалы или нажми \"Сформировать требования\" когда будешь готов.",
		emoji, current, max, progressBar)
}

// renderProgressBar creates a visual progress bar
func renderProgressBar(current, max int) string {
	if max <= 0 {
		return ""
	}

	percent := float64(current) / float64(max)
	filled := int(percent * 10)

	bar := strings.Repeat("▓", filled) + strings.Repeat("░", 10-filled)
	percentage := int(percent * 100)

	return fmt.Sprintf("[%s] %d%%", bar, percentage)
}

// getProgressEmoji returns emoji based on progress
func getProgressEmoji(current, max int) string {
	if max <= 0 {
		return "📥"
	}

	percent := float64(current) / float64(max)
	switch {
	case percent < 0.34:
		return "📥"
	case percent < 0.67:
		return "📊"
	default:
		return "📈"
	}
}

// RenderContextQuestion formats a context question
func RenderContextQuestion(question string) string {
	return fmt.Sprintf(MsgContextQuestion, question)
}

// RenderMaxDraftMessagesError formats max draft messages error
func RenderMaxDraftMessagesError(max int) string {
	return fmt.Sprintf(ErrMaxDraftMessages, max)
}

// EscapeMarkdown escapes special markdown characters
func EscapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// ClassifyError analyzes an error and returns an appropriate user-friendly message
func ClassifyError(err error) string {
	if err == nil {
		return ErrGeneric
	}

	// Check for timeout errors
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ErrTimeout
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ErrTimeout
		}
		return ErrNetworkIssue
	}

	// Check for syscall errors (connection refused, etc.)
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Err == syscall.ECONNREFUSED {
			return ErrServiceUnavailable
		}
		return ErrNetworkIssue
	}

	// Check error message for common patterns
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "connection refused"):
		return ErrServiceUnavailable
	case strings.Contains(errMsg, "timeout"):
		return ErrTimeout
	case strings.Contains(errMsg, "network"):
		return ErrNetworkIssue
	case strings.Contains(errMsg, "unavailable"):
		return ErrServiceUnavailable
	case strings.Contains(errMsg, "quota"):
		return ErrQuotaExceeded
	case strings.Contains(errMsg, "session not found"):
		return ErrSessionNotFound
	case strings.Contains(errMsg, "project not found"):
		return ErrProjectNotFound
	case strings.Contains(errMsg, "transcription failed"), strings.Contains(errMsg, "transcribe"):
		return ErrTranscription
	case strings.Contains(errMsg, "invalid file"), strings.Contains(errMsg, "unsupported format"):
		return ErrInvalidFile
	case strings.Contains(errMsg, "invalid state"):
		return ErrInvalidState
	}

	// Default to generic error
	return ErrGeneric
}
