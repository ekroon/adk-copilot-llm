// Demo script to show slog logging in authentication
// This demonstrates the new structured logging capabilities
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ekroon/adk-copilot-llm/copilot"
)

func main() {
	// Configure slog with different levels to demonstrate logging
	// Set to Debug level to see all logs
	logLevel := os.Getenv("LOG_LEVEL")
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo // Default to INFO
	}

	// Create a text handler with custom options
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	fmt.Println("=== GitHub Copilot Authentication Demo ===")
	fmt.Printf("Log level: %s\n", level.String())
	fmt.Println("Set LOG_LEVEL=DEBUG to see detailed debug logs")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create authenticator
	auth := copilot.NewAuthenticator(copilot.AuthConfig{})

	// Try to authenticate
	// This will show:
	// - INFO: Starting authentication
	// - DEBUG: Device flow details (if LOG_LEVEL=DEBUG)
	// - INFO: Device flow started with user code
	// - INFO: Starting to poll
	// - DEBUG: Each polling attempt (if LOG_LEVEL=DEBUG)
	// - WARN: If slow_down errors occur
	// - INFO: Successful authentication
	token, err := auth.Authenticate(ctx)
	if err != nil {
		fmt.Printf("\nAuthentication failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nAuthentication successful! Token length: %d\n", len(token))
	fmt.Println("\nYou can use this token by setting:")
	fmt.Println("  export GITHUB_TOKEN='...'")
}
