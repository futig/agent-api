package handlers

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os/exec"
	"net/http"
	"net/url"
	"time"

	"bytes"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	maxVoiceFileSize = 10 * 1024 * 1024 // 10 MB
	downloadTimeout  = 30 * time.Second
)

var secureHTTPClient = &http.Client{
	Timeout: downloadTimeout,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	},
}

// downloadVoiceFile is a shared helper for downloading voice files from Telegram
func downloadVoiceFile(ctx context.Context, bot *tgbotapi.BotAPI, fileID string) ([]byte, error) {
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("get file info: %w", err)
	}

	// Check file size before download
	if file.FileSize > maxVoiceFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", file.FileSize, maxVoiceFileSize)
	}

	fileURL := file.Link(bot.Token)

	// Validate URL
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return nil, fmt.Errorf("invalid file URL: %w", err)
	}

	// Ensure HTTPS
	if parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("insecure URL scheme: %s (expected https)", parsedURL.Scheme)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Download file
	resp, err := secureHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read file data with buffered reader for better performance
	// Pre-allocate buffer based on file size
	data := make([]byte, 0, file.FileSize)
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read file data: %w", err)
		}
	}

	// Convert downloaded voice (OGG/Opus) to WAV using ffmpeg
	wavData, err := convertToWav(ctx, data)
	if err != nil {
		return nil, err
	}

	return wavData, nil
}

// convertToWav uses ffmpeg to convert arbitrary audio data (e.g. OGG/Opus from Telegram)
// to mono WAV 16kHz suitable for ASR service.
func convertToWav(ctx context.Context, input []byte) ([]byte, error) {
	// ffmpeg -i pipe:0 -f wav -ar 16000 -ac 1 pipe:1
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", "pipe:0",
		"-f", "wav",
		"-ar", "16000",
		"-ac", "1",
		"pipe:1",
	)

	cmd.Stdin = bytes.NewReader(input)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("ffmpeg convert to wav: %w, stderr: %s", err, stderr.String())
		}
		return nil, fmt.Errorf("ffmpeg convert to wav: %w", err)
	}

	output := stdout.Bytes()
	if len(output) == 0 {
		return nil, fmt.Errorf("ffmpeg convert to wav: empty output")
	}

	return output, nil
}
