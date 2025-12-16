package llm

import (
	"context"
	"fmt"
	"net/http"

	"github.com/futig/agent-backend/internal/config"
	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/integration/common"
	pkghttp "github.com/futig/agent-backend/pkg/http"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type Connector struct {
	config    config.LLMConnectorConfig
	connector *pkghttp.Connector
	logger    *zap.Logger
}

func NewConnector(
	cfg config.LLMConnectorConfig,
	logger *zap.Logger,
) *Connector {
	return &Connector{
		connector: common.NewBaseConnector(cfg.HTTPClientConfig, logger),
		config:    cfg,
		logger:    logger,
	}
}

// GenerateQuestions generates interview questions
func (c *Connector) GenerateQuestions(ctx context.Context, req *entity.LLMGenerateQuestionsRequest) (
	*entity.LLMGenerateQuestionsResponse, error,
) {
	ctxzap.Info(ctx, "generating questions via LLM service")

	var rawResp entity.LLMGenerateQuestionsResponse
	err := c.connector.DoRequest(ctx, http.MethodPost, c.config.GenerateQuestionsEndpoint, req, &rawResp)
	if err != nil {
		return nil, err
	}

	ctxzap.Info(ctx, "questions generated successfully", zap.Int("block_count", len(rawResp.Iterations)))

	return &rawResp, nil
}

// ValidateAnswers validates interview answers
func (c *Connector) ValidateAnswers(ctx context.Context, req *entity.LLMValidateAnswersRequest) (
	*entity.LLMValidateAnswersResponse, error,
) {
	ctxzap.Info(ctx, "validating answers via LLM service")

	var resp entity.LLMValidateAnswersResponse
	err := c.connector.DoRequest(ctx, http.MethodPost, c.config.ValidateAnswersEndpoint, req, &resp)
	if err != nil {
		return nil, fmt.Errorf("validate answers failed: %w", err)
	}

	ctxzap.Info(ctx, "answers validated successfully")

	return &resp, nil
}

// GenerateSummary generates a summary from answers
func (c *Connector) GenerateSummary(ctx context.Context, req *entity.LLMGenerateSummaryRequest) (string, error) {
	ctxzap.Info(ctx, "generating summary via LLM service")

	var resp entity.LLMGenerateSummaryResponse
	err := c.connector.DoRequest(ctx, http.MethodPost, c.config.GenerateSummaryEndpoint, req, &resp)
	if err != nil {
		return "", fmt.Errorf("generate summary failed: %w", err)
	}

	if resp.Result == "" {
		return "", fmt.Errorf("invalid summary response: empty or missing result field")
	}

	ctxzap.Info(ctx, "summary generated successfully", zap.Int("result_length", len(resp.Result)))

	return resp.Result, nil
}

// ValidateDraft validates draft session for rediness to generate final requirements
func (c *Connector) ValidateDraft(ctx context.Context, req *entity.LLMValidateDraftRequest) (
	*entity.LLMValidateAnswersResponse, error,
) {
	ctxzap.Info(ctx, "validating answers via LLM service")

	var resp entity.LLMValidateAnswersResponse
	err := c.connector.DoRequest(ctx, http.MethodPost, c.config.ValidateDraftEndpoint, req, &resp)
	if err != nil {
		return nil, fmt.Errorf("validate answers failed: %w", err)
	}

	ctxzap.Info(ctx, "answers validated successfully")

	return &resp, nil
}

// GenerateDraftSummary generates a summary from draft session
func (c *Connector) GenerateDraftSummary(ctx context.Context, req *entity.LLMGenerateDraftSummaryRequest) (string, error) {
	ctxzap.Info(ctx, "generating summary via LLM service")

	var resp entity.LLMGenerateSummaryResponse
	err := c.connector.DoRequest(ctx, http.MethodPost, c.config.GenerateDraftSummaryEndpoint, req, &resp)
	if err != nil {
		return "", fmt.Errorf("generate summary failed: %w", err)
	}

	if resp.Result == "" {
		return "", fmt.Errorf("invalid summary response: empty or missing result field")
	}

	ctxzap.Info(ctx, "summary generated successfully", zap.Int("result_length", len(resp.Result)))

	return resp.Result, nil
}
