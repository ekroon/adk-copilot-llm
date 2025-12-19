package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ekroon/adk-copilot-llm/copilot"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Example using OAuth device flow
// This token will be exchanged for Copilot API key
func main() {
	ctx := context.Background()

	fmt.Println("GitHub Copilot - OAuth Device Flow Example")
	fmt.Println("===========================================")
	fmt.Println()
	fmt.Println("This example demonstrates using OAuth device flow authentication")
	fmt.Println("The OAuth token will be exchanged for a Copilot API key.")
	fmt.Println()

	// Start device flow authentication
	fmt.Println("Starting OAuth device flow authentication...")
	auth := copilot.NewAuthenticator(copilot.AuthConfig{})
	token, err := auth.Authenticate(ctx)
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	fmt.Println()
	fmt.Printf("Successfully authenticated! Token: %s...\n", token[:20])
	fmt.Println("You can save this token for future use as GITHUB_TOKEN environment variable.")
	fmt.Println()

	// Create Copilot LLM instance
	// The OAuth token will be exchanged for a Copilot API key
	llm, err := copilot.New(copilot.Config{
		GitHubToken: token,
		Model:       "gpt-4",
	})
	if err != nil {
		log.Fatalf("Failed to create Copilot LLM: %v", err)
	}

	fmt.Printf("Using LLM: %s\n\n", llm.Name())

	// Example 1: Code review request
	fmt.Println("Example 1: Code Review")
	fmt.Println("======================")

	request := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Review this Go code and suggest improvements:\n\nfunc add(a, b int) int {\n    return a + b\n}")},
			},
		},
	}

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
	fmt.Println()

	// Example 2: Technical explanation with streaming
	fmt.Println("Example 2: Technical Explanation (Streaming)")
	fmt.Println("============================================")

	streamRequest := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Explain how garbage collection works in Go.")},
			},
		},
	}

	for resp, err := range llm.GenerateContent(ctx, streamRequest, true) {
		if err != nil {
			log.Fatalf("Error generating content: %v", err)
		}
		if resp.Content != nil && len(resp.Content.Parts) > 0 {
			for _, part := range resp.Content.Parts {
				fmt.Print(part.Text)
			}
		}
		if resp.TurnComplete {
			break
		}
	}
	fmt.Println()
	fmt.Println()

	// Example 3: Context-aware conversation
	fmt.Println("Example 3: Context-aware Conversation")
	fmt.Println("=====================================")

	conversationRequest := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("I need to implement rate limiting.")},
			},
			{
				Role:  "model",
				Parts: []*genai.Part{genai.NewPartFromText("I can help you with that! Are you looking to implement rate limiting for an API, or something else?")},
			},
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("For a REST API in Go. What's the best approach?")},
			},
		},
	}

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
	fmt.Println()

	fmt.Println("Note: OAuth tokens are exchanged for time-limited Copilot API keys")
	fmt.Println("that are automatically refreshed when they expire.")
}
