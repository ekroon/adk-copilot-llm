package copilot

import (
	"context"
	"testing"
	"time"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestNew(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		_, err := New(Config{})
		if err == nil {
			t.Error("expected error when GitHubToken is empty")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		llm, err := New(Config{
			GitHubToken: "test-token",
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if llm.Name() != "github-copilot" {
			t.Errorf("expected name 'github-copilot', got %q", llm.Name())
		}
		if llm.config.Model != "gpt-4" {
			t.Errorf("expected default model 'gpt-4', got %q", llm.config.Model)
		}
	})

	t.Run("enterprise config", func(t *testing.T) {
		llm, err := New(Config{
			GitHubToken:   "test-token",
			EnterpriseURL: "company.ghe.com",
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		expectedBaseURL := "https://copilot-api.company.ghe.com"
		if llm.baseURL != expectedBaseURL {
			t.Errorf("expected baseURL %q, got %q", expectedBaseURL, llm.baseURL)
		}
	})

	t.Run("custom model", func(t *testing.T) {
		llm, err := New(Config{
			GitHubToken: "test-token",
			Model:       "gpt-3.5-turbo",
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if llm.config.Model != "gpt-3.5-turbo" {
			t.Errorf("expected model 'gpt-3.5-turbo', got %q", llm.config.Model)
		}
	})
}

func TestConvertRequest(t *testing.T) {
	llm := &CopilotLLM{}

	t.Run("single text part", func(t *testing.T) {
		req := &model.LLMRequest{
			Contents: []*genai.Content{
				{
					Role:  "user",
					Parts: []*genai.Part{genai.NewPartFromText("Hello")},
				},
			},
		}

		chatReq, err := llm.convertRequest(req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(chatReq.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(chatReq.Messages))
		}
		if chatReq.Messages[0].Content != "Hello" {
			t.Errorf("expected content 'Hello', got %q", chatReq.Messages[0].Content)
		}
		if chatReq.Messages[0].Role != "user" {
			t.Errorf("expected role 'user', got %q", chatReq.Messages[0].Role)
		}
	})

	t.Run("multiple contents", func(t *testing.T) {
		req := &model.LLMRequest{
			Contents: []*genai.Content{
				{
					Role:  "user",
					Parts: []*genai.Part{genai.NewPartFromText("Hello")},
				},
				{
					Role:  "model",
					Parts: []*genai.Part{genai.NewPartFromText("Hi there!")},
				},
			},
		}

		chatReq, err := llm.convertRequest(req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(chatReq.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(chatReq.Messages))
		}
	})

	t.Run("with config", func(t *testing.T) {
		temp := float32(0.8)
		topP := float32(0.9)
		maxTokens := int32(100)

		req := &model.LLMRequest{
			Contents: []*genai.Content{
				{
					Role:  "user",
					Parts: []*genai.Part{genai.NewPartFromText("Hello")},
				},
			},
			Config: &genai.GenerateContentConfig{
				Temperature:     &temp,
				TopP:            &topP,
				MaxOutputTokens: maxTokens,
			},
		}

		chatReq, err := llm.convertRequest(req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if chatReq.Temperature == nil {
			t.Error("expected temperature to be set")
		} else {
			diff := *chatReq.Temperature - 0.8
			if diff < -0.001 || diff > 0.001 {
				t.Errorf("expected temperature 0.8, got %f", *chatReq.Temperature)
			}
		}
		if chatReq.TopP == nil {
			t.Error("expected topP to be set")
		} else {
			diff := *chatReq.TopP - 0.9
			if diff < -0.001 || diff > 0.001 {
				t.Errorf("expected topP 0.9, got %f", *chatReq.TopP)
			}
		}
		if chatReq.MaxTokens == nil {
			t.Error("expected maxTokens to be set")
		} else if *chatReq.MaxTokens != 100 {
			t.Errorf("expected maxTokens 100, got %d", *chatReq.MaxTokens)
		}
	})
}

func TestMapFinishReason(t *testing.T) {
	tests := []struct {
		input    string
		expected genai.FinishReason
	}{
		{"stop", genai.FinishReasonStop},
		{"length", genai.FinishReasonMaxTokens},
		{"content_filter", genai.FinishReasonSafety},
		{"unknown", genai.FinishReasonOther},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapFinishReason(tt.input)
			if result != tt.expected {
				t.Errorf("mapFinishReason(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsPAT(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "valid PAT token",
			token:    "github_pat_1234567890abcdef",
			expected: true,
		},
		{
			name:     "OAuth token",
			token:    "gho_1234567890abcdef",
			expected: false,
		},
		{
			name:     "empty token",
			token:    "",
			expected: false,
		},
		{
			name:     "random string",
			token:    "random_token_string",
			expected: false,
		},
		{
			name:     "prefix without underscore",
			token:    "github_pat",
			expected: false,
		},
		{
			name:     "PAT prefix in middle",
			token:    "prefix_github_pat_suffix",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPAT(tt.token)
			if result != tt.expected {
				t.Errorf("isPAT(%q) = %v; want %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestEnsureAPIKeyWithPAT(t *testing.T) {
	t.Run("PAT token used directly", func(t *testing.T) {
		patToken := "github_pat_1234567890abcdef"
		llm, err := New(Config{
			GitHubToken: patToken,
		})
		if err != nil {
			t.Fatalf("failed to create LLM: %v", err)
		}

		ctx := context.Background()
		err = llm.ensureAPIKey(ctx)
		if err != nil {
			t.Errorf("ensureAPIKey failed: %v", err)
		}

		// Verify PAT was set directly
		llm.mu.RLock()
		apiKey := llm.copilotAPIKey
		expiresAt := llm.apiKeyExpiresAt
		llm.mu.RUnlock()

		if apiKey != patToken {
			t.Errorf("expected copilotAPIKey to be PAT token %q, got %q", patToken, apiKey)
		}

		// Verify expiration is far in the future (at least 5 years)
		fiveYearsFromNow := time.Now().Add(5 * 365 * 24 * time.Hour)
		if expiresAt.Before(fiveYearsFromNow) {
			t.Errorf("expected expiration to be far in future, got %v", expiresAt)
		}
	})

	t.Run("PAT token cached on subsequent calls", func(t *testing.T) {
		patToken := "github_pat_cached_test"
		llm, err := New(Config{
			GitHubToken: patToken,
		})
		if err != nil {
			t.Fatalf("failed to create LLM: %v", err)
		}

		ctx := context.Background()

		// First call
		err = llm.ensureAPIKey(ctx)
		if err != nil {
			t.Errorf("first ensureAPIKey call failed: %v", err)
		}

		llm.mu.RLock()
		firstAPIKey := llm.copilotAPIKey
		firstExpiry := llm.apiKeyExpiresAt
		llm.mu.RUnlock()

		// Second call - should use cached key
		err = llm.ensureAPIKey(ctx)
		if err != nil {
			t.Errorf("second ensureAPIKey call failed: %v", err)
		}

		llm.mu.RLock()
		secondAPIKey := llm.copilotAPIKey
		secondExpiry := llm.apiKeyExpiresAt
		llm.mu.RUnlock()

		if firstAPIKey != secondAPIKey {
			t.Errorf("API key changed between calls: %q != %q", firstAPIKey, secondAPIKey)
		}

		if firstExpiry != secondExpiry {
			t.Errorf("expiration changed between calls: %v != %v", firstExpiry, secondExpiry)
		}
	})
}

func TestChatMessageMarshalJSON(t *testing.T) {
	t.Run("string content", func(t *testing.T) {
		msg := chatMessage{
			Role:    "user",
			Content: "Hello",
		}
		data, err := msg.MarshalJSON()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		expected := `{"role":"user","content":"Hello"}`
		if string(data) != expected {
			t.Errorf("expected %q, got %q", expected, string(data))
		}
	})

	t.Run("array content", func(t *testing.T) {
		msg := chatMessage{
			Role: "user",
			ContentParts: []map[string]interface{}{
				{"type": "text", "text": "Hello"},
			},
		}
		data, err := msg.MarshalJSON()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Just check it doesn't error and contains expected fields
		str := string(data)
		if str == "" {
			t.Error("expected non-empty JSON")
		}
	})
}
