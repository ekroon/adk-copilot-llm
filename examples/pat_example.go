package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ekroon/adk-copilot-llm/copilot"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Example using Personal Access Token (github_pat_*)
// This token is used directly without exchange
func main() {
	ctx := context.Background()

	// Replace with your GitHub Personal Access Token
	// Get it from: GitHub Settings → Developer settings → Personal access tokens
	token := "github_pat_YOUR_TOKEN_HERE"

	fmt.Println("GitHub Copilot - Personal Access Token Example")
	fmt.Println("===============================================")
	fmt.Println()
	fmt.Println("This example demonstrates using a GitHub Personal Access Token (PAT)")
	fmt.Println("PAT tokens start with 'github_pat_' and are used directly without token exchange.")
	fmt.Println()

	// Create Copilot LLM instance with PAT token
	llm, err := copilot.New(copilot.Config{
		GitHubToken: token,
		Model:       "gpt-4",
	})
	if err != nil {
		log.Fatalf("Failed to create Copilot LLM: %v", err)
	}

	fmt.Printf("Using LLM: %s\n\n", llm.Name())

	// Example 1: Simple question
	fmt.Println("Example 1: Simple Question")
	fmt.Println("==========================")

	request := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What are the key benefits of using Go for backend development?")},
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

	// Example 2: Code generation with streaming
	fmt.Println("Example 2: Code Generation (Streaming)")
	fmt.Println("======================================")

	streamRequest := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Write a Go function to calculate fibonacci numbers recursively.")},
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
	fmt.Println("Example 3: Multi-turn Conversation")
	fmt.Println("===================================")

	conversationRequest := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("I'm working on a REST API in Go.")},
			},
			{
				Role:  "model",
				Parts: []*genai.Part{genai.NewPartFromText("That's great! Go is excellent for building REST APIs. What specific aspect would you like help with?")},
			},
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What's the best way to handle errors in API responses?")},
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

	fmt.Println("Note: PAT tokens are used directly without token exchange,")
	fmt.Println("making them ideal for quick setup and development.")
}
