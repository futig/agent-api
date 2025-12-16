package entity

import (
	"mime/multipart"
)

type ResultFormat string

const (
	FormatMarkdown ResultFormat = "markdown"
	FormatDOCX     ResultFormat = "docx"
	FormatPDF      ResultFormat = "pdf"
)

func (f ResultFormat) IsValid() bool {
	switch f {
	case FormatMarkdown, FormatDOCX, FormatPDF:
		return true
	default:
		return false
	}
}

type CreateProjectRequest struct {
	Title       string
	Description string
	Files       []*multipart.FileHeader
	CallbackURL string
}

type CreateProjectResponse struct {
	Status    string `json:"status"`
	ProjectID string `json:"project_id"`
}

type ListProjectsRequest struct {
	Skip  int
	Limit int
}

func (lp *ListProjectsRequest) Normalize() {
	if lp.Limit <= 0 {
		lp.Limit = 10
	}

	lp.Limit = min(lp.Limit, 100)
}

type ListProjectsResponse struct {
	Projects []*ProjectSummary `json:"projects"`
}

type ProjectSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type ProjectDetailResponse struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Size        int64         `json:"size"`
	Files       []*FileDetail `json:"files"`
}

type FileDetail struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

type DeleteProjectResponse struct {
	Status string `json:"status"`
}

type AddFilesRequest struct {
	ProjectID   string
	Files       []*multipart.FileHeader
	CallbackURL string
}

type AddFilesResponse struct {
	Status string `json:"status"`
}

type ListFilesResponse struct {
	Files []*FileDetail `json:"files"`
}
