package main

import (
	"log"

	"github.com/futig/agent-backend/internal/builder"
)

func main() {
	app, err := builder.Build()
	if err != nil {
		log.Fatal("Failed to build application:", err)
	}

	if err := app.Run(); err != nil {
		log.Fatal("Application error:", err)
	}
}
