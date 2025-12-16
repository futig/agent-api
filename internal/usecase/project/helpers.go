package project

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/pkg/validator"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// prepareFileData reads file contents and prepares them for RAG indexing
func (uc *ProjectUsecase) prepareFileData(
	ctx context.Context,
	files []*multipart.FileHeader,
) ([]entity.FileData, error) {
	fileDataList := make([]entity.FileData, 0, len(files))

	for _, fh := range files {
		src, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("open file %s: %w", fh.Filename, err)
		}

		content, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", fh.Filename, err)
		}

		fileDataList = append(fileDataList, entity.FileData{
			Filename: validator.SanitizeFilename(fh.Filename),
			Content:  content,
		})

		ctxzap.Debug(ctx, "file prepared for indexing",
			zap.String("filename", fh.Filename),
			zap.Int64("size", fh.Size),
		)
	}

	return fileDataList, nil
}

// saveFileMetadata saves file metadata to database after successful RAG indexing
func (uc *ProjectUsecase) saveFileMetadata(
	ctx context.Context,
	projectID string,
	files []*multipart.FileHeader,
) ([]*entity.File, error) {
	savedFiles := make([]*entity.File, 0, len(files))

	for _, fh := range files {
		fileID := uuid.New().String()

		file := &entity.File{
			ID:          fileID,
			ProjectID:   projectID,
			Filename:    validator.SanitizeFilename(fh.Filename),
			Size:        fh.Size,
			ContentType: fh.Header.Get("Content-Type"),
		}

		savedFile, err := uc.projectFileRepo.AddFile(ctx, *file)
		if err != nil {
			uc.cleanupFileMetadata(ctx, uc.extractFileIDs(savedFiles))
			return nil, fmt.Errorf("save file metadata for %s: %w", fh.Filename, err)
		}
		savedFiles = append(savedFiles, savedFile)

		ctxzap.Info(ctx, "file metadata saved",
			zap.String("project_id", projectID),
			zap.String("file_id", fileID),
			zap.String("filename", fh.Filename),
		)
	}

	return savedFiles, nil
}

// extractFileIDs extracts file IDs from a slice of files
func (uc *ProjectUsecase) extractFileIDs(files []*entity.File) []string {
	ids := make([]string, len(files))
	for i, f := range files {
		ids[i] = f.ID
	}
	return ids
}

// cleanupFileMetadata removes file metadata from database
func (uc *ProjectUsecase) cleanupFileMetadata(ctx context.Context, fileIDs []string) {
	for _, fileID := range fileIDs {
		if err := uc.projectFileRepo.DeleteFile(ctx, fileID); err != nil {
			ctxzap.Warn(ctx, "failed to delete file metadata during cleanup",
				zap.String("file_id", fileID),
				zap.Error(err),
			)
		}
	}
}
