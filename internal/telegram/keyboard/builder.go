package keyboard

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Builder creates inline keyboards
type Builder struct{}

// NewBuilder creates a keyboard builder
func NewBuilder() *Builder {
	return &Builder{}
}

// StartKeyboard creates the initial start button
func (b *Builder) StartKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Начать сессию", "action:start"),
		),
	)
}

// ModeSelectionKeyboard creates Interview/Draft selection buttons
func (b *Builder) ModeSelectionKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Интервью", "mode:interview"),
			tgbotapi.NewInlineKeyboardButtonData("📄 Драфт", "mode:draft"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Сменить проект", "action:change_project"),
		),
	)
}

// ProjectSelectionKeyboard creates project selection buttons
func (b *Builder) ProjectSelectionKeyboard(projects []Project) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{}

	// Add project buttons (max 10 recent)
	count := len(projects)
	if count > 10 {
		count = 10
	}

	for i := 0; i < count; i++ {
		proj := projects[i]
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				proj.Title,
				"proj:"+proj.ID,
			),
		))
	}

	// Add "No project" button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ Проекта нет", "proj:none"),
	))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// ProjectSelectionKeyboardWithPagination creates project selection buttons with pagination
func (b *Builder) ProjectSelectionKeyboardWithPagination(projects []Project, hasPrev, hasNext bool) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{}

	// Add project buttons
	for _, proj := range projects {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				proj.Title,
				"proj:"+proj.ID,
			),
		))
	}

	// Add "No project" button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ Проекта нет", "proj:none"),
	))

	// Add pagination buttons if needed
	if hasPrev || hasNext {
		navRow := []tgbotapi.InlineKeyboardButton{}
		if hasPrev {
			navRow = append(navRow,
				tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "page:prev"))
		}
		if hasNext {
			navRow = append(navRow,
				tgbotapi.NewInlineKeyboardButtonData("Вперёд ▶️", "page:next"))
		}
		rows = append(rows, navRow)
	}

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// QuestionNavigationKeyboard creates question navigation buttons
func (b *Builder) QuestionNavigationKeyboard(questionID string, hasPrevious bool) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "skip:"+questionID),
			tgbotapi.NewInlineKeyboardButtonData("❓ Поясни вопрос", "explain:"+questionID),
		),
	}

	// Add back button if there are previous questions
	if hasPrevious {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Предыдущий вопрос", "prev:"+questionID),
		))
	}

	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Сформировать требования", "action:generate"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛑 Завершить диалог", "action:finish"),
		),
	)

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// InterviewInfoKeyboard creates interview info confirmation buttons
func (b *Builder) InterviewInfoKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, начать интервью", "action:start_interview"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Выбрать другой формат", "action:choose_mode"),
		),
	)
}

// DraftInfoKeyboard creates draft info confirmation buttons
func (b *Builder) DraftInfoKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, начать", "action:start_draft"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Выбрать другой формат", "action:choose_mode"),
		),
	)
}

// DraftCollectionKeyboard creates draft collection control buttons
func (b *Builder) DraftCollectionKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Сформировать требования", "action:generate"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛑 Закрыть сессию", "action:finish"),
		),
	)
}

// ResultSaveKeyboard creates result save and download buttons
func (b *Builder) ResultSaveKeyboard(hasSkipped bool, projectTitle string) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💾 Сохранить в новый проект", "action:save_new_project"),
		),
	}

	// Add "Save to existing project" button only if projectTitle is provided
	if projectTitle != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("💾 Сохранить в '%s'", projectTitle), "action:save_to_project"),
		))
	}

	// Download buttons
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📄 Скачать .md", "dl:markdown"),
		tgbotapi.NewInlineKeyboardButtonData("📕 Скачать .pdf", "dl:pdf"),
	))

	if hasSkipped {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Ответить на пропущенные", "action:answer_skipped"),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ Завершить диалог", "action:finish"),
	))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// ResultDownloadKeyboard creates result download buttons (deprecated, use ResultSaveKeyboard)
func (b *Builder) ResultDownloadKeyboard(hasSkipped bool) tgbotapi.InlineKeyboardMarkup {
	return b.ResultSaveKeyboard(hasSkipped, "")
}

// ResultDownloadOnlyKeyboard creates download buttons without save options (after project is already saved)
func (b *Builder) ResultDownloadOnlyKeyboard(hasSkipped bool) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📄 Скачать .md", "dl:markdown"),
			tgbotapi.NewInlineKeyboardButtonData("📕 Скачать .pdf", "dl:pdf"),
		),
	}

	if hasSkipped {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Ответить на пропущенные", "action:answer_skipped"),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ Завершить диалог", "action:finish"),
	))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// Project represents a project for keyboard building
type Project struct {
	ID    string
	Title string
}
