package project

import "github.com/futig/agent-backend/internal/entity"

// toProjectSummary converts Project entity to ProjectSummary DTO
func toProjectSummary(p *entity.Project) *entity.ProjectSummary {
	return &entity.ProjectSummary{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
	}
}

// toProjectDetail converts Project entity to ProjectDetailResponse DTO
func toProjectDetail(p *entity.Project) *entity.ProjectDetailResponse {
	var size int64
	files := make([]*entity.FileDetail, 0, len(p.Files))
	for _, f := range p.Files {
		size += f.Size
		files = append(files, toFileDetail(f))
	}

	return &entity.ProjectDetailResponse{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		Files:       files,
		Size:        size,
	}
}

// toFileDetail converts File entity to FileDetail DTO
func toFileDetail(f *entity.File) *entity.FileDetail {
	return &entity.FileDetail{
		ID:        f.ID,
		Name:      f.Filename,
		Size:      f.Size,
		CreatedAt: f.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// toCallbackProjectUpdated converts Project entity to CallbackProjectUpdatedData
func toCallbackProjectUpdated(p *entity.Project) *entity.CallbackProjectUpdatedData {
	var totalSize int64
	fileInfos := make([]entity.CallbackFileInfo, len(p.Files))
	for i, f := range p.Files {
		totalSize += f.Size
		fileInfos[i] = entity.CallbackFileInfo{
			ID:   f.ID,
			Name: f.Filename,
			Size: f.Size,
		}
	}

	return &entity.CallbackProjectUpdatedData{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		Size:        totalSize,
		Files:       fileInfos,
	}
}
