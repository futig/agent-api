package middleware

import (
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// userLimit tracks rate limit state for a single user
type userLimit struct {
	tokens        float64
	lastRefill    time.Time
	warningsSent  int
	lastWarningAt time.Time
	mu            sync.Mutex
}

// RateLimiterMiddleware implements token bucket rate limiting per user
type RateLimiterMiddleware struct {
	limits          map[int64]*userLimit
	mu              sync.RWMutex
	maxTokens       float64 // Maximum tokens in bucket
	refillRate      float64 // Tokens added per second
	burstSize       int     // Max burst size
	warningInterval time.Duration
	logger          *zap.Logger
	api             *tgbotapi.BotAPI
}

// NewRateLimiterMiddleware creates a new rate limiter middleware
func NewRateLimiterMiddleware(
	requestsPerMinute int,
	burstSize int,
	logger *zap.Logger,
	api *tgbotapi.BotAPI,
) *RateLimiterMiddleware {
	rl := &RateLimiterMiddleware{
		limits:          make(map[int64]*userLimit),
		maxTokens:       float64(requestsPerMinute),
		refillRate:      float64(requestsPerMinute) / 60.0, // tokens per second
		burstSize:       burstSize,
		warningInterval: 30 * time.Second,
		logger:          logger,
		api:             api,
	}

	// Start cleanup goroutine to remove inactive users
	go rl.cleanupInactiveUsers()

	return rl
}

// Handle processes the update through rate limiting
func (rl *RateLimiterMiddleware) Handle(update tgbotapi.Update, next func(tgbotapi.Update)) {
	var userID int64
	var chatID int64

	// Extract user and chat ID
	if update.Message != nil {
		userID = update.Message.From.ID
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
		chatID = update.CallbackQuery.Message.Chat.ID
	} else {
		// Unknown update type, allow it
		next(update)
		return
	}

	// Check rate limit
	if !rl.allowRequest(userID, chatID) {
		rl.logger.Warn("rate limit exceeded",
			zap.Int64("user_id", userID),
			zap.Int64("chat_id", chatID),
		)
		return
	}

	// Call next handler
	next(update)
}

// allowRequest checks if request is allowed under rate limit
func (rl *RateLimiterMiddleware) allowRequest(userID, chatID int64) bool {
	rl.mu.Lock()
	limit, exists := rl.limits[userID]
	if !exists {
		limit = &userLimit{
			tokens:     rl.maxTokens,
			lastRefill: time.Now(),
		}
		rl.limits[userID] = limit
	}
	rl.mu.Unlock()

	limit.mu.Lock()
	defer limit.mu.Unlock()

	now := time.Now()

	// Refill tokens based on elapsed time
	elapsed := now.Sub(limit.lastRefill).Seconds()
	limit.tokens += elapsed * rl.refillRate
	if limit.tokens > rl.maxTokens {
		limit.tokens = rl.maxTokens
	}
	limit.lastRefill = now

	// Check if we have enough tokens
	if limit.tokens >= 1.0 {
		limit.tokens -= 1.0
		limit.warningsSent = 0 // Reset warnings on successful request
		return true
	}

	// Rate limit exceeded - send warning if not sent recently
	if now.Sub(limit.lastWarningAt) > rl.warningInterval {
		limit.warningsSent++
		limit.lastWarningAt = now

		rl.sendRateLimitWarning(chatID, limit.warningsSent)
	}

	return false
}

// sendRateLimitWarning sends a warning message to the user
func (rl *RateLimiterMiddleware) sendRateLimitWarning(chatID int64, warningCount int) {
	var text string

	switch {
	case warningCount == 1:
		text = "⚠️ Слишком много запросов. Пожалуйста, подождите немного."
	case warningCount == 2:
		text = "⚠️ Превышен лимит запросов. Подождите ~30 секунд перед следующей попыткой."
	case warningCount >= 3:
		text = "🛑 Вы отправляете запросы слишком часто. Пожалуйста, подождите минуту."
	}

	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := rl.api.Send(msg); err != nil {
		rl.logger.Error("failed to send rate limit warning",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
		)
	}
}

// cleanupInactiveUsers removes users that haven't sent requests in 1 hour
func (rl *RateLimiterMiddleware) cleanupInactiveUsers() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		inactiveThreshold := 1 * time.Hour

		for userID, limit := range rl.limits {
			limit.mu.Lock()
			if now.Sub(limit.lastRefill) > inactiveThreshold {
				delete(rl.limits, userID)
				rl.logger.Debug("cleaned up inactive user from rate limiter",
					zap.Int64("user_id", userID),
				)
			}
			limit.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}
