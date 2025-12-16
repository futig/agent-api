package validator

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/futig/agent-backend/internal/entity"
)

// ValidateStartSession validates StartSessionRequest
func (v *Validator) ValidateStartSession(req *entity.StartSessionRequest) error {
	if req.UserGoal == "" {
		return fmt.Errorf("%w: context_goal", entity.ErrMissingField)
	}

	if req.CallbackURL == "" {
		return fmt.Errorf("%w: callback_url", entity.ErrMissingField)
	}

	if (req.ProjectID == nil || *req.ProjectID == "") && len(req.ContextQuestions) == 0 {
		return fmt.Errorf("project_id and context_questions must not be both empty at the same time")
	}

	if (req.ProjectID != nil && *req.ProjectID != "") && len(req.ContextQuestions) != 0 {
		return fmt.Errorf("project_id and context_questions must not be both filled at the same time")
	}

	return nil
}

// ValidateSubmitAnswer validates answer submission
func (v *Validator) ValidateSubmitAnswer(req *entity.SubmitAnswerRequest) error {
	if req.CallbackURL == "" {
		return fmt.Errorf("%w: callback_url", entity.ErrMissingField)
	}
	if !req.IsSkipped && req.Answer == "" {
		return fmt.Errorf("%w: answers", entity.ErrMissingField)
	}

	return nil
}

// ValidateSubmitAudioAnswer validates audio answer submission
func (v *Validator) ValidateSubmitAudioAnswer(req *entity.SubmitAudioAnswerRequest) error {
	if req.CallbackURL == "" {
		return fmt.Errorf("%w: callback_url", entity.ErrMissingField)
	}
	if !req.IsSkipped && req.AudioFile == nil {
		return fmt.Errorf("%w: audio file", entity.ErrMissingField)
	}

	if req.AudioFile != nil {
		return v.ValidateAudioFile(req.AudioFile)
	}

	return nil
}

// ValidateAudioFile validates audio file uploads (WAV format only)
func (v *Validator) ValidateAudioFile(file *multipart.FileHeader) error {
	if file == nil {
		return entity.ErrMissingField
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".wav" {
		return fmt.Errorf("%w: %s (only .wav files are allowed)", entity.ErrInvalidExtension, ext)
	}

	// Check file size
	if file.Size > v.cfg.MaxAudioFileSize {
		return fmt.Errorf("%w: file '%s' is %d bytes (max %d)", entity.ErrFileTooLarge, file.Filename, file.Size, v.cfg.MaxAudioFileSize)
	}

	// Check content type if provided
	contentType := file.Header.Get("Content-Type")
	if contentType != "" &&
		contentType != "audio/wav" &&
		contentType != "audio/x-wav" &&
		contentType != "application/octet-stream" {
		return fmt.Errorf("%w: content type '%s' (expected audio/wav, audio/x-wav or application/octet-stream)", entity.ErrInvalidExtension, contentType)
	}

	return nil
}
