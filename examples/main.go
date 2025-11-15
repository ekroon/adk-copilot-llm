package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ekroon/adk-copilot-llm/copilot"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	// Option 1: Use existing GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")

	// Option 2: If no token, perform device flow authentication
	if token == "" {
		fmt.Println("No GITHUB_TOKEN found, starting device flow authentication...")
		auth := copilot.NewAuthenticator(copilot.AuthConfig{})
		var err error
		token, err = auth.Authenticate(ctx)
		if err != nil {
			log.Fatalf("Authentication failed: %v", err)
		}
		fmt.Printf("\nYou can set this token as GITHUB_TOKEN environment variable for future use.\n\n")
	}

	// Create Copilot LLM instance
	llm, err := copilot.New(copilot.Config{
		GitHubToken: token,
		Model:       "gpt-4", // or "gpt-3.5-turbo"
	})
	if err != nil {
		log.Fatalf("Failed to create Copilot LLM: %v", err)
	}

	fmt.Printf("Using LLM: %s\n\n", llm.Name())

	// Example 1: Non-streaming request
	fmt.Println("Example 1: Non-streaming request")
	fmt.Println("==================================")

	request := &model.LLMRequest{
		Model: "gpt-4",
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What is the capital of France?")},
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

	// Example 2: Streaming request
	fmt.Println("Example 2: Streaming request")
	fmt.Println("=============================")

	streamRequest := &model.LLMRequest{
		Model: "gpt-4",
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Write a short poem about coding.")},
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

	// Example 3: Multi-turn conversation
	fmt.Println("Example 3: Multi-turn conversation")
	fmt.Println("===================================")

	conversationRequest := &model.LLMRequest{
		Model: "gpt-4",
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
}
