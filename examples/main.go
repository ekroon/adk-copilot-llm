// Package main demonstrates the usage of the adk-copilot-llm package
// which provides an implementation of the adk-go LLM interface for GitHub Copilot.
//
// This example shows:
// - Creating a CopilotLLM with minimal configuration
// - Non-streaming requests
// - Streaming requests
// - Multi-turn conversations
//
// Prerequisites:
// - The Copilot CLI must be installed and available in PATH (or set COPILOT_CLI_PATH)
// - You must be authenticated with GitHub Copilot (the CLI handles this automatically)
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ekroon/adk-copilot-llm/copilot"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	// =========================================================================
	// Create the CopilotLLM instance
	// =========================================================================
	// The copilot CLI handles all authentication automatically.
	// Only the model needs to be specified; other options have sensible defaults.
	llm, err := copilot.New(copilot.Config{
		Model: "gpt-4", // or "gpt-3.5-turbo", "claude-3.5-sonnet", etc.
	})
	if err != nil {
		log.Fatalf("Failed to create Copilot LLM: %v", err)
	}
	defer llm.Close() // Ensure proper cleanup when done

	fmt.Printf("Using LLM: %s\n\n", llm.Name())

	// =========================================================================
	// Example 1: Non-streaming request
	// =========================================================================
	// A simple question-and-answer with the complete response returned at once.
	fmt.Println("Example 1: Non-streaming request")
	fmt.Println("=================================")

	request := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What is the capital of France?")},
			},
		},
	}

	// Iterate over responses (non-streaming returns one complete response)
	for resp, err := range llm.GenerateContent(ctx, request, false) {
		if err != nil {
			log.Fatalf("Error generating content: %v", err)
		}
		if resp.Content != nil && len(resp.Content.Parts) > 0 {
			for _, part := range resp.Content.Parts {
				fmt.Print(part.Text)
			}
		}
	}
	fmt.Println()

	// =========================================================================
	// Example 2: Streaming request
	// =========================================================================
	// Streaming returns partial responses as they are generated,
	// allowing for real-time display of content.
	fmt.Println("Example 2: Streaming request")
	fmt.Println("============================")

	streamRequest := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Write a short poem about coding.")},
			},
		},
	}

	// Iterate over streaming responses - chunks arrive as they're generated
	for resp, err := range llm.GenerateContent(ctx, streamRequest, true) {
		if err != nil {
			log.Fatalf("Error generating content: %v", err)
		}
		// Print each chunk as it arrives
		if resp.Content != nil && len(resp.Content.Parts) > 0 {
			for _, part := range resp.Content.Parts {
				fmt.Print(part.Text)
			}
		}
		// TurnComplete signals the end of the response
		if resp.TurnComplete {
			break
		}
	}
	fmt.Println()

	// =========================================================================
	// Example 3: Multi-turn conversation
	// =========================================================================
	// Demonstrates context preservation across multiple turns.
	// The conversation history is passed in the Contents array.
	fmt.Println("Example 3: Multi-turn conversation")
	fmt.Println("===================================")

	conversationRequest := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("My name is Alice.")},
			},
			{
				Role:  "model",
				Parts: []*genai.Part{genai.NewPartFromText("Hello Alice! Nice to meet you. How can I help you today?")},
			},
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What's my name?")},
			},
		},
	}

	// The model should remember "Alice" from the conversation history
	for resp, err := range llm.GenerateContent(ctx, conversationRequest, false) {
		if err != nil {
			log.Fatalf("Error generating content: %v", err)
		}
		if resp.Content != nil && len(resp.Content.Parts) > 0 {
			for _, part := range resp.Content.Parts {
				fmt.Print(part.Text)
			}
		}
	}
	fmt.Println()

	fmt.Println("All examples completed successfully!")
}
