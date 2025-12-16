package asr

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// MockConnector - мок-реализация ASR коннектора для тестирования
type MockConnector struct {
	logger *zap.Logger
}

func NewMockConnector(logger *zap.Logger) *MockConnector {
	return &MockConnector{
		logger: logger,
	}
}

// TranscribeBytes - мок транскрибации аудио
func (m *MockConnector) TranscribeBytes(ctx context.Context, audioData []byte, filename string) (string, error) {
	if len(audioData) == 0 {
		return "", fmt.Errorf("empty audio data provided")
	}

	ctxzap.Info(ctx, "[MOCK] transcribing audio via ASR",
		zap.String("filename", filename),
		zap.Int("size", len(audioData)),
	)

	// Возвращаем мок-транскрипцию
	mockTranscription := `Добрый день. Я хочу рассказать о требованиях к нашей системе.
Во-первых, необходимо реализовать функциональность регистрации и авторизации пользователей.
Во-вторых, система должна поддерживать работу с заказами и их обработку.
Также важно обеспечить интеграцию с внешними платежными системами для приема оплаты.
Производительность системы должна быть достаточной для обработки до тысячи одновременных пользователей.
Все данные должны храниться в защищенном виде с использованием современных методов шифрования.`

	ctxzap.Info(ctx, "[MOCK] audio transcribed", zap.Int("transcription_length", len(mockTranscription)))
	return mockTranscription, nil
}
