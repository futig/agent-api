package rag

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/futig/agent-backend/internal/config"
	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/integration/common"
	pkghttp "github.com/futig/agent-backend/pkg/http"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type Connector struct {
	config    config.RAGConnectorConfig
	connector *pkghttp.Connector
	logger    *zap.Logger
}

func NewConnector(
	cfg config.RAGConnectorConfig,
	logger *zap.Logger,
) *Connector {
	return &Connector{
		connector: common.NewBaseConnector(cfg.HTTPClientConfig, logger),
		config:    cfg,
		logger:    logger,
	}
}

// IndexFiles indexes files for a project
// POST {index_endpoint}?project_id={id} with multipart/form-data
func (c *Connector) IndexFiles(ctx context.Context, projectID string, files []entity.FileData) error {
	endpoint := fmt.Sprintf("%s?project_id=%s", c.config.IndexEndpoint, projectID)

	ctxzap.Info(ctx, "indexing files in RAG service", zap.Int("file_count", len(files)))

	prepareBody := func(writer *multipart.Writer) error {
		for _, file := range files {
			part, err := writer.CreateFormFile("files", file.Filename)
			if err != nil {
				return fmt.Errorf("create form file: %w", err)
			}

			if _, err := part.Write(file.Content); err != nil {
				return fmt.Errorf("write file content: %w", err)
			}
		}
		return nil
	}

	err := c.connector.DoMultipartRequest(ctx, http.MethodPost, endpoint, prepareBody, nil)
	if err != nil {
		ctxzap.Error(ctx, "failed to index files", zap.Error(err))
		return err
	}

	ctxzap.Info(ctx, "files indexed successfully")
	return nil
}

// DeleteIndex deletes the index for a project
// DELETE {delete_endpoint} with {project_id} substituted
// Returns true on any 2xx, false on HTTP error/exception
func (c *Connector) DeleteIndex(ctx context.Context, projectID string) error {
	endpoint := strings.Replace(c.config.DeleteEndpoint, "{project_id}", projectID, 1)

	ctxzap.Info(ctx, "deleting RAG index")

	var resp entity.RAGDeleteIndexResponse
	err := c.connector.DoRequest(ctx, http.MethodDelete, endpoint, nil, &resp)
	if err != nil {
		ctxzap.Error(ctx, "failed to delete index", zap.Error(err))
		return err
	}

	ctxzap.Info(ctx, "index deleted successfully", zap.Int("deleted_count", resp.DeletedCount))
	return nil
}

// GetContext retrieves relevant context from RAG service
func (c *Connector) GetContext(ctx context.Context, req *entity.RAGGetContextRequest) (string, error) {
	ctxzap.Debug(ctx, "getting context from RAG service")

	var resp entity.RAGGetContextResponse
	err := c.connector.DoRequest(ctx, http.MethodPost, c.config.ContextEndpoint, req, &resp)
	if err != nil {
		return "", fmt.Errorf("failed to get context: %w", err)
	}

	// Extract and join text from relevant chunks
	var texts []string
	for _, chunk := range resp.RelevantContext.RelevantChunks {
		if chunk.Text != "" {
			texts = append(texts, chunk.Text)
		}
	}

	result := strings.Join(texts, "\n\n")
	ctxzap.Debug(ctx, "context retrieved",
		zap.Int("chunk_count", len(texts)),
		zap.Int("total_length", len(result)),
	)

	return result, nil
}
