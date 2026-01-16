package copilot

import (
	"os"
	"testing"

	"google.golang.org/genai"
)

func TestNew(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		// Clear env var to test default
		originalEnv := os.Getenv("COPILOT_CLI_PATH")
		os.Unsetenv("COPILOT_CLI_PATH")
		defer func() {
			if originalEnv != "" {
				os.Setenv("COPILOT_CLI_PATH", originalEnv)
			}
		}()

		llm, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if llm.config.Model != "gpt-4" {
			t.Errorf("expected default model 'gpt-4', got %q", llm.config.Model)
		}
		if llm.config.LogLevel != "error" {
			t.Errorf("expected default log level 'error', got %q", llm.config.LogLevel)
		}
		if llm.config.CLIPath != "copilot" {
			t.Errorf("expected default CLIPath 'copilot', got %q", llm.config.CLIPath)
		}
	})

	t.Run("COPILOT_CLI_PATH env var", func(t *testing.T) {
		originalEnv := os.Getenv("COPILOT_CLI_PATH")
		os.Setenv("COPILOT_CLI_PATH", "/custom/path/copilot")
		defer func() {
			if originalEnv != "" {
				os.Setenv("COPILOT_CLI_PATH", originalEnv)
			} else {
				os.Unsetenv("COPILOT_CLI_PATH")
			}
		}()

		llm, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if llm.config.CLIPath != "/custom/path/copilot" {
			t.Errorf("expected CLIPath from env '/custom/path/copilot', got %q", llm.config.CLIPath)
		}
	})

	t.Run("custom model", func(t *testing.T) {
		llm, err := New(Config{
			Model: "gpt-3.5-turbo",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if llm.config.Model != "gpt-3.5-turbo" {
			t.Errorf("expected model 'gpt-3.5-turbo', got %q", llm.config.Model)
		}
	})

	t.Run("custom CLIPath overrides env", func(t *testing.T) {
		originalEnv := os.Getenv("COPILOT_CLI_PATH")
		os.Setenv("COPILOT_CLI_PATH", "/env/path/copilot")
		defer func() {
			if originalEnv != "" {
				os.Setenv("COPILOT_CLI_PATH", originalEnv)
			} else {
				os.Unsetenv("COPILOT_CLI_PATH")
			}
		}()

		llm, err := New(Config{
			CLIPath: "/explicit/path/copilot",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if llm.config.CLIPath != "/explicit/path/copilot" {
			t.Errorf("expected CLIPath '/explicit/path/copilot', got %q", llm.config.CLIPath)
		}
	})

	t.Run("custom log level", func(t *testing.T) {
		llm, err := New(Config{
			LogLevel: "debug",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if llm.config.LogLevel != "debug" {
			t.Errorf("expected log level 'debug', got %q", llm.config.LogLevel)
		}
	})

	t.Run("streaming config", func(t *testing.T) {
		llm, err := New(Config{
			Streaming: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !llm.config.Streaming {
			t.Error("expected streaming to be true")
		}
	})

	t.Run("client not started initially", func(t *testing.T) {
		llm, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if llm.started {
			t.Error("expected client to not be started initially")
		}
		if llm.client == nil {
			t.Error("expected client to be created")
		}
	})
}

func TestName(t *testing.T) {
	llm, err := New(Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if llm.Name() != "github-copilot" {
		t.Errorf("expected name 'github-copilot', got %q", llm.Name())
	}
}

func TestFormatPrompt(t *testing.T) {
	t.Run("empty contents", func(t *testing.T) {
		result := formatPrompt(nil)
		if result != "" {
			t.Errorf("expected empty string for nil contents, got %q", result)
		}

		result = formatPrompt([]*genai.Content{})
		if result != "" {
			t.Errorf("expected empty string for empty contents, got %q", result)
		}
	})

	t.Run("single content", func(t *testing.T) {
		contents := []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Hello, world!")},
			},
		}

		result := formatPrompt(contents)
		if result != "Hello, world!" {
			t.Errorf("expected 'Hello, world!', got %q", result)
		}
	})

	t.Run("multi-turn conversation with user", func(t *testing.T) {
		contents := []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Hello")},
			},
			{
				Role:  "model",
				Parts: []*genai.Part{genai.NewPartFromText("Hi there!")},
			},
		}

		result := formatPrompt(contents)
		expected := "User: Hello\n\nAssistant: Hi there!"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("model role mapped to Assistant", func(t *testing.T) {
		contents := []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Question")},
			},
			{
				Role:  "model",
				Parts: []*genai.Part{genai.NewPartFromText("Answer")},
			},
		}

		result := formatPrompt(contents)
		expected := "User: Question\n\nAssistant: Answer"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("assistant role", func(t *testing.T) {
		contents := []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Question")},
			},
			{
				Role:  "assistant",
				Parts: []*genai.Part{genai.NewPartFromText("Answer")},
			},
		}

		result := formatPrompt(contents)
		expected := "User: Question\n\nAssistant: Answer"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("system role", func(t *testing.T) {
		contents := []*genai.Content{
			{
				Role:  "system",
				Parts: []*genai.Part{genai.NewPartFromText("You are helpful")},
			},
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Hello")},
			},
		}

		result := formatPrompt(contents)
		expected := "System: You are helpful\n\nUser: Hello"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("unknown role uses role name", func(t *testing.T) {
		contents := []*genai.Content{
			{
				Role:  "custom",
				Parts: []*genai.Part{genai.NewPartFromText("Custom message")},
			},
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Hello")},
			},
		}

		result := formatPrompt(contents)
		expected := "custom: Custom message\n\nUser: Hello"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("skips empty content", func(t *testing.T) {
		contents := []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Hello")},
			},
			{
				Role:  "model",
				Parts: []*genai.Part{}, // Empty parts
			},
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Follow up")},
			},
		}

		result := formatPrompt(contents)
		expected := "User: Hello\n\nUser: Follow up"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("case insensitive roles", func(t *testing.T) {
		contents := []*genai.Content{
			{
				Role:  "USER",
				Parts: []*genai.Part{genai.NewPartFromText("Hello")},
			},
			{
				Role:  "MODEL",
				Parts: []*genai.Part{genai.NewPartFromText("Hi")},
			},
		}

		result := formatPrompt(contents)
		expected := "User: Hello\n\nAssistant: Hi"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestExtractText(t *testing.T) {
	t.Run("nil content", func(t *testing.T) {
		result := extractText(nil)
		if result != "" {
			t.Errorf("expected empty string for nil content, got %q", result)
		}
	})

	t.Run("empty parts", func(t *testing.T) {
		content := &genai.Content{
			Role:  "user",
			Parts: []*genai.Part{},
		}

		result := extractText(content)
		if result != "" {
			t.Errorf("expected empty string for empty parts, got %q", result)
		}
	})

	t.Run("nil parts", func(t *testing.T) {
		content := &genai.Content{
			Role:  "user",
			Parts: nil,
		}

		result := extractText(content)
		if result != "" {
			t.Errorf("expected empty string for nil parts, got %q", result)
		}
	})

	t.Run("single text part", func(t *testing.T) {
		content := &genai.Content{
			Role:  "user",
			Parts: []*genai.Part{genai.NewPartFromText("Hello, world!")},
		}

		result := extractText(content)
		if result != "Hello, world!" {
			t.Errorf("expected 'Hello, world!', got %q", result)
		}
	})

	t.Run("multiple text parts", func(t *testing.T) {
		content := &genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText("First part"),
				genai.NewPartFromText("Second part"),
			},
		}

		result := extractText(content)
		expected := "First part\nSecond part"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("parts with empty text skipped", func(t *testing.T) {
		content := &genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText("First"),
				genai.NewPartFromText(""),
				genai.NewPartFromText("Third"),
			},
		}

		result := extractText(content)
		expected := "First\nThird"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestClose(t *testing.T) {
	t.Run("close unstarted client", func(t *testing.T) {
		llm, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error creating LLM: %v", err)
		}

		// Ensure client is not started
		if llm.started {
			t.Fatal("expected client to not be started")
		}

		// Close should not error on unstarted client
		err = llm.Close()
		if err != nil {
			t.Errorf("unexpected error closing unstarted client: %v", err)
		}

		// Verify started is still false
		if llm.started {
			t.Error("expected started to remain false after closing unstarted client")
		}
	})

	t.Run("close is idempotent", func(t *testing.T) {
		llm, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error creating LLM: %v", err)
		}

		// Close multiple times should not error
		for i := 0; i < 3; i++ {
			err = llm.Close()
			if err != nil {
				t.Errorf("unexpected error on close attempt %d: %v", i+1, err)
			}
		}
	})
}
