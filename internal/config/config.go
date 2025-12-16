package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/caarlos0/env/v11"
	pkgRetry "github.com/futig/agent-backend/internal/pkg/retry"
	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	ServerAddr string `env:"SERVER_ADDR,notEmpty"`

	// Database configuration
	DatabaseURL         string        `env:"DATABASE_URL,notEmpty"`
	DBMaxConns          int           `env:"DB_MAX_CONNS" envDefault:"25"`
	DBMinConns          int           `env:"DB_MIN_CONNS" envDefault:"5"`
	DBMaxConnLifetime   time.Duration `env:"DB_MAX_CONN_LIFETIME" envDefault:"1h"`
	DBMaxConnIdleTime   time.Duration `env:"DB_MAX_CONN_IDLE_TIME" envDefault:"30m"`
	DBHealthCheckPeriod time.Duration `env:"DB_HEALTH_CHECK_PERIOD" envDefault:"1m"`

	// External service configurations
	RAGConnectorCfg      RAGConnectorConfig      `envPrefix:"RAG_"`
	LLMConnectorCfg      LLMConnectorConfig      `envPrefix:"LLM_"`
	ASRConnectorCfg      ASRConnectorConfig      `envPrefix:"ASR_"`
	CallbackConnectorCfg CallbackConnectorConfig `envPrefix:"CALLBACK_"`

	// Logging configuration
	LogLevel string `env:"LOG_LEVEL,notEmpty"`

	// File upload configuration
	FileUploadCfg FileUploadConfig `envPrefix:"FILE_UPLOAD_"`

	// Context questions configuration (loaded from JSON file)
	ContextQuestions []string

	// Mock configuration
	EnableMocks bool `env:"ENABLE_MOCKS,notEmpty"`

	// Telegram bot configuration (optional)
	TelegramCfg TelegramConfig `envPrefix:"TELEGRAM_"`

	// Environment (set from flag, not from env var)
	Environment string
}

// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	BotToken              string `env:"BOT_TOKEN,notEmpty"`
	WebhookURL            string `env:"WEBHOOK_URL,notEmpty"`
	UseWebhook            bool   `env:"USE_WEBHOOK,notEmpty"`
	UpdateTimeout         int    `env:"UPDATE_TIMEOUT,notEmpty"`
	MaxConcurrentUsers    int    `env:"MAX_CONCURRENT_USERS,notEmpty"`
	MaxDraftMessages      int    `env:"MAX_DRAFT_MESSAGES,notEmpty"`
	RateLimitPerMinute    int    `env:"RATE_LIMIT_PER_MINUTE,notEmpty"`
	RateLimitBurst        int    `env:"RATE_LIMIT_BURST,notEmpty"`
	ShutdownTimeout       int    `env:"SHUTDOWN_TIMEOUT,notEmpty"` // seconds
}

type RAGConnectorConfig struct {
	HTTPClientConfig
	IndexEndpoint   string               `env:"INDEX_ENDPOINT,notEmpty"`
	DeleteEndpoint  string               `env:"DELETE_ENDPOINT,notEmpty"`
	ContextEndpoint string               `env:"CONTEXT_ENDPOINT,notEmpty"`
	Retry           pkgRetry.RetryConfig `envPrefix:"RETRY_"`
}

type LLMConnectorConfig struct {
	HTTPClientConfig
	GenerateQuestionsEndpoint    string               `env:"GENERATE_QUESTIONS_ENDPOINT,notEmpty"`
	ValidateAnswersEndpoint      string               `env:"VALIDATE_ANSWERS_ENDPOINT,notEmpty"`
	GenerateSummaryEndpoint      string               `env:"GENERATE_SUMMARY_ENDPOINT,notEmpty"`
	ValidateDraftEndpoint        string               `env:"VALIDATE_DRAFT_ENDPOINT,notEmpty"`
	GenerateDraftSummaryEndpoint string               `env:"GENERATE_DRAFT_SUMMARY_ENDPOINT,notEmpty"`
	Retry                        pkgRetry.RetryConfig `envPrefix:"RETRY_"`
}

type ASRConnectorConfig struct {
	HTTPClientConfig
	TranscribeEndpoint string               `env:"TRANSCRIBE_ENDPOINT,notEmpty"`
	Retry              pkgRetry.RetryConfig `envPrefix:"RETRY_"`
}

type CallbackConnectorConfig struct {
	HTTPClientConfig
	CallbackEndpoint string               `env:"ENDPOINT,notEmpty"`
	Retry            pkgRetry.RetryConfig `envPrefix:"RETRY_"`
}

type HTTPClientConfig struct {
	RequestTimeout        time.Duration `env:"TIMEOUT,notEmpty"`
	ConnTimeout           time.Duration `env:"CONN_TIMEOUT,notEmpty"`
	KeepAlive             time.Duration `env:"KEEP_ALIVE,notEmpty"`
	IdleConnTimeout       time.Duration `env:"IDLE_CONN_TIMEOUT,notEmpty"`
	ResponseHeaderTimeout time.Duration `env:"RESPONSE_HEADER_TIMEOUT,notEmpty"`
	Token                 string        `env:"TOKEN"`
	Url                   string        `env:"SERVICE_URL,notEmpty"`
}

// FileUploadConfig holds file upload limits
type FileUploadConfig struct {
	MaxFileSize      int64 `env:"MAX_FILE_SIZE,notEmpty"`       // 5 MiB
	MaxTotalSize     int64 `env:"MAX_TOTAL_SIZE,notEmpty"`      // 25 MiB
	MaxFileCount     int   `env:"MAX_FILE_COUNT,notEmpty"`      // Max 64 files
	MaxAudioFileSize int64 `env:"MAX_AUDIO_FILE_SIZE,notEmpty"` // 25 MiB
	MaxUploadSize    int64 `env:"MAX_UPLOAD_SIZE,notEmpty"`     // 32 MB
}

// contextQuestions represents the structure of context_questions.json
type contextQuestions struct {
	Questions []string `json:"questions"`
}

func LoadConfig() (*Config, error) {
	envFlag := flag.String("env", "local", "Environment to run (local, prod, or custom)")
	flag.Parse()

	envFile := getEnvFile(*envFlag)
	// Try to load env file, but don't fail if it's missing.
	// In containerized/prod environments variables are usually set externally.
	if err := godotenv.Load(envFile); err != nil {
		fmt.Printf("Warning: could not load %s file (this is ok if env vars are set externally): %v\n", envFile, err)
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	cfg.Environment = *envFlag

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Load context questions from JSON file
	if err := loadContextQuestions(cfg); err != nil {
		return nil, fmt.Errorf("load context questions: %w", err)
	}

	return cfg, nil
}

func validateConfig(cfg *Config) error {
	var errors []string

	// Validate Telegram configuration
	if cfg.TelegramCfg.MaxDraftMessages < 1 || cfg.TelegramCfg.MaxDraftMessages > 50 {
		errors = append(errors, fmt.Sprintf("MAX_DRAFT_MESSAGES must be between 1 and 50, got %d", cfg.TelegramCfg.MaxDraftMessages))
	}

	if cfg.TelegramCfg.RateLimitPerMinute < 1 || cfg.TelegramCfg.RateLimitPerMinute > 60 {
		errors = append(errors, fmt.Sprintf("TELEGRAM_RATE_LIMIT_PER_MINUTE must be between 1 and 60, got %d", cfg.TelegramCfg.RateLimitPerMinute))
	}

	if cfg.TelegramCfg.RateLimitBurst < 1 || cfg.TelegramCfg.RateLimitBurst > 20 {
		errors = append(errors, fmt.Sprintf("TELEGRAM_RATE_LIMIT_BURST must be between 1 and 20, got %d", cfg.TelegramCfg.RateLimitBurst))
	}

	if cfg.TelegramCfg.ShutdownTimeout < 1 || cfg.TelegramCfg.ShutdownTimeout > 300 {
		errors = append(errors, fmt.Sprintf("TELEGRAM_SHUTDOWN_TIMEOUT must be between 1 and 300 seconds, got %d", cfg.TelegramCfg.ShutdownTimeout))
	}

	// Validate Database configuration
	if cfg.DBMaxConns < 1 || cfg.DBMaxConns > 200 {
		errors = append(errors, fmt.Sprintf("DB_MAX_CONNS must be between 1 and 200, got %d", cfg.DBMaxConns))
	}

	if cfg.DBMinConns < 0 || cfg.DBMinConns > cfg.DBMaxConns {
		errors = append(errors, fmt.Sprintf("DB_MIN_CONNS must be between 0 and DB_MAX_CONNS(%d), got %d", cfg.DBMaxConns, cfg.DBMinConns))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation errors:\n  - %s", fmt.Sprintf("%s", errors[0]))
	}

	return nil
}

var defaultContextQuestions = []string{
	"Опишите цель проекта",
	"Кто основные пользователи системы?",
	"Какие основные функции должна выполнять система?",
	"Есть ли интеграции с внешними системами?",
	"Какие технические ограничения существуют?",
}

func loadContextQuestions(cfg *Config) error {
	configDir := filepath.Join("internal", "config", "context_questions.json")

	// Check if file exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		fmt.Printf("Warning: context questions file not found at %s, using default questions\n", configDir)
		cfg.ContextQuestions = defaultContextQuestions
		return nil
	}

	data, err := os.ReadFile(configDir)
	if err != nil {
		return fmt.Errorf("read context questions file: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("context questions file is empty: %s", configDir)
	}

	var questionsData contextQuestions
	if err := json.Unmarshal(data, &questionsData); err != nil {
		return fmt.Errorf("parse context questions JSON: %w", err)
	}

	if len(questionsData.Questions) == 0 {
		return fmt.Errorf("context questions file contains no questions: %s", configDir)
	}

	cfg.ContextQuestions = questionsData.Questions

	fmt.Printf("Loaded %d context questions from %s\n", len(cfg.ContextQuestions), configDir)
	return nil
}

func getEnvFile(environment string) string {
	switch environment {
	case "prod", "production":
		return ".env.prod"
	case "local", "dev", "development":
		return ".env.local"
	default:
		return fmt.Sprintf(".env.%s", environment)
	}
}
