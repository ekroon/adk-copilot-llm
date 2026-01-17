package copilot

import (
	"context"
	"os"
	"reflect"
	"testing"

	"google.golang.org/adk/tool"
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

func TestToolConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		tools   []tool.Tool
		wantErr bool
	}{
		{
			name:    "nil tools",
			tools:   nil,
			wantErr: false,
		},
		{
			name:    "empty tools",
			tools:   []tool.Tool{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm, err := New(Config{
				Model: "gpt-4",
				Tools: tt.tools,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if llm != nil {
				defer llm.Close()
			}
		})
	}
}

func TestToolContextImplementation(t *testing.T) {
	ctx := context.Background()
	tc := &toolContext{
		ctx:    ctx,
		callID: "test-call-123",
	}

	// Test Context()
	if tc.Context() != ctx {
		t.Errorf("Context() returned wrong context")
	}

	// Test FunctionCallID()
	if got := tc.FunctionCallID(); got != "test-call-123" {
		t.Errorf("FunctionCallID() = %q, want %q", got, "test-call-123")
	}

	// Test nil returns for agent-runtime features
	if tc.Agent() != nil {
		t.Errorf("Agent() should return nil in standalone mode")
	}
	if tc.Session() != nil {
		t.Errorf("Session() should return nil in standalone mode")
	}
	if tc.Actions() != nil {
		t.Errorf("Actions() should return nil in standalone mode")
	}

	// Test SearchMemory returns error
	_, err := tc.SearchMemory(ctx, "test query")
	if err == nil {
		t.Errorf("SearchMemory() should return error in standalone mode")
	}
}

func TestDeclarationToParams(t *testing.T) {
	tests := []struct {
		name string
		decl *genai.FunctionDeclaration
		want map[string]interface{}
	}{
		{
			name: "nil declaration",
			decl: &genai.FunctionDeclaration{},
			want: nil,
		},
		{
			name: "with ParametersJsonSchema",
			decl: &genai.FunctionDeclaration{
				ParametersJsonSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
					},
				},
			},
			want: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := declarationToParams(tt.decl)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("declarationToParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSchemaToMap(t *testing.T) {
	t.Run("nil schema", func(t *testing.T) {
		result := schemaToMap(nil)
		if result != nil {
			t.Errorf("expected nil result, got %v", result)
		}
	})

	t.Run("empty schema", func(t *testing.T) {
		result := schemaToMap(&genai.Schema{})
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	t.Run("simple string schema", func(t *testing.T) {
		schema := &genai.Schema{
			Type:        genai.TypeString,
			Description: "A string field",
		}

		result := schemaToMap(schema)
		if result["type"] != "string" {
			t.Errorf("expected type 'string', got %v", result["type"])
		}
		if result["description"] != "A string field" {
			t.Errorf("expected description 'A string field', got %v", result["description"])
		}
	})

	t.Run("object schema with properties", func(t *testing.T) {
		schema := &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"name": {
					Type:        genai.TypeString,
					Description: "Name field",
				},
				"age": {
					Type:        genai.TypeInteger,
					Description: "Age field",
				},
			},
			Required: []string{"name"},
		}

		result := schemaToMap(schema)
		if result["type"] != "object" {
			t.Errorf("expected type 'object', got %v", result["type"])
		}

		props, ok := result["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("expected properties to be map[string]interface{}")
		}
		if len(props) != 2 {
			t.Errorf("expected 2 properties, got %d", len(props))
		}

		required, ok := result["required"].([]string)
		if !ok {
			t.Fatal("expected required to be []string")
		}
		if len(required) != 1 || required[0] != "name" {
			t.Errorf("expected required=['name'], got %v", required)
		}
	})

	t.Run("array schema", func(t *testing.T) {
		schema := &genai.Schema{
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeString,
			},
		}

		result := schemaToMap(schema)
		if result["type"] != "array" {
			t.Errorf("expected type 'array', got %v", result["type"])
		}

		items, ok := result["items"].(map[string]interface{})
		if !ok {
			t.Fatal("expected items to be map[string]interface{}")
		}
		if items["type"] != "string" {
			t.Errorf("expected items type 'string', got %v", items["type"])
		}
	})

	t.Run("enum schema", func(t *testing.T) {
		schema := &genai.Schema{
			Type: genai.TypeString,
			Enum: []string{"option1", "option2", "option3"},
		}

		result := schemaToMap(schema)
		enum, ok := result["enum"].([]string)
		if !ok {
			t.Fatal("expected enum to be []string")
		}
		if len(enum) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(enum))
		}
	})

	t.Run("numeric constraints", func(t *testing.T) {
		min := 0.0
		max := 100.0
		schema := &genai.Schema{
			Type:    genai.TypeNumber,
			Minimum: &min,
			Maximum: &max,
		}

		result := schemaToMap(schema)
		if result["minimum"] != 0.0 {
			t.Errorf("expected minimum 0.0, got %v", result["minimum"])
		}
		if result["maximum"] != 100.0 {
			t.Errorf("expected maximum 100.0, got %v", result["maximum"])
		}
	})

	t.Run("string constraints", func(t *testing.T) {
		minLen := int64(1)
		maxLen := int64(50)
		schema := &genai.Schema{
			Type:      genai.TypeString,
			MinLength: &minLen,
			MaxLength: &maxLen,
			Pattern:   "^[a-z]+$",
			Format:    "email",
		}

		result := schemaToMap(schema)
		if result["minLength"] != int64(1) {
			t.Errorf("expected minLength 1, got %v", result["minLength"])
		}
		if result["maxLength"] != int64(50) {
			t.Errorf("expected maxLength 50, got %v", result["maxLength"])
		}
		if result["pattern"] != "^[a-z]+$" {
			t.Errorf("expected pattern '^[a-z]+$', got %v", result["pattern"])
		}
		if result["format"] != "email" {
			t.Errorf("expected format 'email', got %v", result["format"])
		}
	})
}
