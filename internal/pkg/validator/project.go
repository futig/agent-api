package validator

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/futig/agent-backend/internal/config"
	"github.com/futig/agent-backend/internal/entity"
)

var AllowedExtensions = map[string]bool{
	".txt":  true,
	".md":   true,
	".docx": true,
}

// Validator validates file uploads
type Validator struct {
	cfg config.FileUploadConfig
}

func NewFileValidator(cfg config.FileUploadConfig) *Validator {
	return &Validator{cfg: cfg}
}

func (v *Validator) ValidateCreateProject(req *entity.CreateProjectRequest) error {
	if req.Title == "" {
		return fmt.Errorf("%w: title", entity.ErrMissingField)
	}
	if req.Description == "" {
		return fmt.Errorf("%w: description", entity.ErrMissingField)
	}
	if req.CallbackURL == "" {
		return fmt.Errorf("%w: callback_url", entity.ErrMissingField)
	}
	if len(req.Files) == 0 {
		return fmt.Errorf("%w: files", entity.ErrMissingField)
	}

	return v.ValidateUpload(req.Files)
}

// ValidateUpload validates multiple file uploads
func (v *Validator) ValidateUpload(files []*multipart.FileHeader) error {
	if len(files) == 0 {
		return entity.ErrMissingField
	}

	if len(files) > int(v.cfg.MaxFileCount) {
		return fmt.Errorf("%w: maximum %d files allowed, got %d", entity.ErrTooManyFiles, v.cfg.MaxFileCount, len(files))
	}

	var totalSize int64
	for _, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if _, ok := AllowedExtensions[ext]; !ok {
			return fmt.Errorf("%w: %s (allowed: txt, md, docx)", entity.ErrInvalidExtension, ext)
		}

		if fh.Size > v.cfg.MaxFileSize {
			return fmt.Errorf("%w: file '%s' is %d bytes (max %d)", entity.ErrFileTooLarge, fh.Filename, fh.Size, v.cfg.MaxFileSize)
		}

		totalSize += fh.Size
	}

	if totalSize > v.cfg.MaxTotalSize {
		return fmt.Errorf("%w: total size is %d bytes (max %d)", entity.ErrTotalSizeTooLarge, totalSize, v.cfg.MaxTotalSize)
	}

	return nil
}

// SanitizeFilename sanitizes a filename for safe storage
func SanitizeFilename(filename string) string {
	filename = filepath.Base(filename)
	replacer := strings.NewReplacer(
		" ", "_",
		"(", "",
		")", "",
		"[", "",
		"]", "",
		"{", "",
		"}", "",
	)
	return replacer.Replace(filename)
}
