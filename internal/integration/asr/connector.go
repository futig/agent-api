package asr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/futig/agent-backend/internal/config"
	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/integration/common"
	pkghttp "github.com/futig/agent-backend/pkg/http"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type Connector struct {
	config    config.ASRConnectorConfig
	connector *pkghttp.Connector
	logger    *zap.Logger
}

func NewConnector(
	cfg config.ASRConnectorConfig,
	logger *zap.Logger,
) *Connector {
	return &Connector{
		connector: common.NewBaseConnector(cfg.HTTPClientConfig, logger),
		config:    cfg,
		logger:    logger,
	}
}

// transcribeBytes is the internal method for transcribing audio bytes
func (c *Connector) TranscribeBytes(ctx context.Context, audioData []byte, filename string) (string, error) {
	if len(audioData) == 0 {
		return "", fmt.Errorf("empty audio data provided")
	}

	hash := sha256.Sum256(audioData)
	checksum := hex.EncodeToString(hash[:])

	ctxzap.Info(ctx, "transcribing audio via ASR service",
		zap.String("filename", filename),
		zap.String("checksum", checksum),
		zap.Int("size", len(audioData)),
	)

	prepareBody := func(writer *multipart.Writer) error {
		// Add file part
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return fmt.Errorf("create form file: %w", err)
		}

		if _, err := part.Write(audioData); err != nil {
			return fmt.Errorf("write file content: %w", err)
		}

		// Add checksum field
		if err := writer.WriteField("checksum", checksum); err != nil {
			return fmt.Errorf("write checksum field: %w", err)
		}

		return nil
	}

	var resp entity.ASRTranscribeResponse
	err := c.connector.DoMultipartRequest(ctx, http.MethodPost, c.config.TranscribeEndpoint, prepareBody, &resp)
	if err != nil {
		return "", fmt.Errorf("failed to transcribe audio: %w", err)
	}

	ctxzap.Info(ctx, "audio transcribed successfully", zap.Int("transcription_length", len(resp.Transcriptions)))

	return resp.Transcriptions, nil
}
