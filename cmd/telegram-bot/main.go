package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/futig/agent-backend/internal/builder"
	"go.uber.org/zap"
)

func main() {
	bot, logger, err := builder.BuildTelegramBot()
	if err != nil {
		log.Fatal("Failed to build telegram bot:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		logger.Info("starting telegram bot...")
		if err := bot.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	select {
	case sig := <-sigChan:
		logger.Info("received shutdown signal",
			zap.String("signal", sig.String()))
		cancel()
		if err := bot.Stop(); err != nil {
			logger.Error("error stopping bot",
				zap.Error(err))
		}
		logger.Info("telegram bot stopped gracefully")
	case err := <-errChan:
		logger.Error("telegram bot error",
			zap.Error(err))
		cancel()
		os.Exit(1)
	}
}
