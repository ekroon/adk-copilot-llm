package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ekroon/adk-copilot-llm/copilot"
	"github.com/zalando/go-keyring"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

const (
	// Service name for keyring storage
	keyringService = "adk-copilot-llm"
	// User name for keyring storage
	keyringUser = "github-token"
)

func main() {
	ctx := context.Background()

	// Try to get token from keyring
	token, err := getTokenFromKeyring()
	if err != nil {
		fmt.Println("No token found in keyring, starting authentication flow...")
		token, err = authenticateAndStore(ctx)
		if err != nil {
			log.Fatalf("Authentication failed: %v", err)
		}
		fmt.Println("Token stored securely in keyring for future use.")
	} else {
		fmt.Println("Using token from keyring.")
	}

	// Create Copilot LLM instance
	llm, err := copilot.New(copilot.Config{
		GitHubToken: token,
		Model:       "gpt-4",
	})
	if err != nil {
		log.Fatalf("Failed to create Copilot LLM: %v", err)
	}

	fmt.Printf("Using LLM: %s\n\n", llm.Name())

	// Example request
	fmt.Println("Example: Asking a question")
	fmt.Println("===========================")

	request := &model.LLMRequest{
		Model: "gpt-4",
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What is GitHub Copilot?")},
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
	fmt.Println("Example completed successfully!")
}

// getTokenFromKeyring retrieves the GitHub token from the system keyring.
func getTokenFromKeyring() (string, error) {
	token, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return "", err
	}
	return token, nil
}

// setTokenInKeyring stores the GitHub token in the system keyring.
func setTokenInKeyring(token string) error {
	return keyring.Set(keyringService, keyringUser, token)
}

// authenticateAndStore performs device flow authentication and stores the token.
func authenticateAndStore(ctx context.Context) (string, error) {
	// Start device flow authentication
	auth := copilot.NewAuthenticator(copilot.AuthConfig{})
	token, err := auth.Authenticate(ctx)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	// Store token in keyring
	if err := setTokenInKeyring(token); err != nil {
		// Log the error but don't fail - token can still be used
		fmt.Printf("Warning: Failed to store token in keyring: %v\n", err)
	}

	return token, nil
}

// DeleteToken is a helper function to remove the token from keyring.
// This can be called if you want to re-authenticate or clear stored credentials.
func DeleteToken() error {
	return keyring.Delete(keyringService, keyringUser)
}
