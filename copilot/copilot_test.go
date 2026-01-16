package copilot

import (
	"fmt"
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

func TestConvertTools(t *testing.T) {
	t.Run("nil tools", func(t *testing.T) {
		llm, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := llm.convertTools(nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d tools", len(result))
		}
	})

	t.Run("empty tools", func(t *testing.T) {
		llm, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := llm.convertTools([]*genai.Tool{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d tools", len(result))
		}
	})

	t.Run("tool without function declarations", func(t *testing.T) {
		llm, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		genaiTools := []*genai.Tool{
			{
				FunctionDeclarations: nil,
			},
		}

		result, err := llm.convertTools(genaiTools)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d tools", len(result))
		}
	})

	t.Run("tool without handler", func(t *testing.T) {
		llm, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		genaiTools := []*genai.Tool{
			{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{
						Name:        "test_tool",
						Description: "A test tool",
					},
				},
			},
		}

		_, err = llm.convertTools(genaiTools)
		if err == nil {
			t.Error("expected error for tool without handler")
		}
		if err != nil && err.Error() != "no handler found for tool: test_tool" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("single tool with handler", func(t *testing.T) {
		handlerCalled := false
		llm, err := New(Config{
			ToolHandlers: map[string]ToolHandler{
				"test_tool": func(args map[string]any) (string, error) {
					handlerCalled = true
					return "test result", nil
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		genaiTools := []*genai.Tool{
			{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{
						Name:        "test_tool",
						Description: "A test tool",
					},
				},
			},
		}

		result, err := llm.convertTools(genaiTools)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result))
		}

		tool := result[0]
		if tool.Name != "test_tool" {
			t.Errorf("expected name 'test_tool', got %q", tool.Name)
		}
		if tool.Description != "A test tool" {
			t.Errorf("expected description 'A test tool', got %q", tool.Description)
		}
		if tool.Handler == nil {
			t.Fatal("expected handler to be set")
		}

		// Test handler wasn't called during conversion
		if handlerCalled {
			t.Error("handler should not be called during conversion")
		}
	})

	t.Run("tool with parameters", func(t *testing.T) {
		llm, err := New(Config{
			ToolHandlers: map[string]ToolHandler{
				"calculator": func(args map[string]any) (string, error) {
					return "42", nil
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		genaiTools := []*genai.Tool{
			{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{
						Name:        "calculator",
						Description: "Perform calculations",
						Parameters: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"operation": {
									Type:        genai.TypeString,
									Description: "The operation to perform",
									Enum:        []string{"add", "subtract"},
								},
								"a": {
									Type:        genai.TypeNumber,
									Description: "First number",
								},
								"b": {
									Type:        genai.TypeNumber,
									Description: "Second number",
								},
							},
							Required: []string{"operation", "a", "b"},
						},
					},
				},
			},
		}

		result, err := llm.convertTools(genaiTools)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result))
		}

		tool := result[0]
		if tool.Parameters == nil {
			t.Fatal("expected parameters to be set")
		}

		// Verify type
		if tool.Parameters["type"] != "object" {
			t.Errorf("expected type 'object', got %v", tool.Parameters["type"])
		}

		// Verify properties
		props, ok := tool.Parameters["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("expected properties to be map[string]interface{}")
		}
		if len(props) != 3 {
			t.Errorf("expected 3 properties, got %d", len(props))
		}

		// Verify required fields
		required, ok := tool.Parameters["required"].([]string)
		if !ok {
			t.Fatal("expected required to be []string")
		}
		if len(required) != 3 {
			t.Errorf("expected 3 required fields, got %d", len(required))
		}
	})

	t.Run("multiple tools", func(t *testing.T) {
		llm, err := New(Config{
			ToolHandlers: map[string]ToolHandler{
				"tool1": func(args map[string]any) (string, error) {
					return "result1", nil
				},
				"tool2": func(args map[string]any) (string, error) {
					return "result2", nil
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		genaiTools := []*genai.Tool{
			{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{
						Name:        "tool1",
						Description: "First tool",
					},
					{
						Name:        "tool2",
						Description: "Second tool",
					},
				},
			},
		}

		result, err := llm.convertTools(genaiTools)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(result))
		}

		if result[0].Name != "tool1" || result[1].Name != "tool2" {
			t.Error("tools not in expected order")
		}
	})
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

func TestToolHandlerRegistration(t *testing.T) {
	// Test that tool handlers can be registered in Config
	t.Run("single handler registration", func(t *testing.T) {
		handler := func(args map[string]any) (string, error) {
			return "test result", nil
		}

		llm, err := New(Config{
			ToolHandlers: map[string]ToolHandler{
				"test_tool": handler,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if llm.config.ToolHandlers == nil {
			t.Fatal("expected ToolHandlers to be set")
		}
		if len(llm.config.ToolHandlers) != 1 {
			t.Errorf("expected 1 handler, got %d", len(llm.config.ToolHandlers))
		}
		if _, ok := llm.config.ToolHandlers["test_tool"]; !ok {
			t.Error("expected test_tool handler to be registered")
		}
	})

	t.Run("multiple handler registration", func(t *testing.T) {
		handler1 := func(args map[string]any) (string, error) {
			return "result1", nil
		}
		handler2 := func(args map[string]any) (string, error) {
			return "result2", nil
		}
		handler3 := func(args map[string]any) (string, error) {
			return "result3", nil
		}

		llm, err := New(Config{
			ToolHandlers: map[string]ToolHandler{
				"tool1": handler1,
				"tool2": handler2,
				"tool3": handler3,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(llm.config.ToolHandlers) != 3 {
			t.Errorf("expected 3 handlers, got %d", len(llm.config.ToolHandlers))
		}
		for _, toolName := range []string{"tool1", "tool2", "tool3"} {
			if _, ok := llm.config.ToolHandlers[toolName]; !ok {
				t.Errorf("expected %s handler to be registered", toolName)
			}
		}
	})

	t.Run("nil handlers map", func(t *testing.T) {
		llm, err := New(Config{
			ToolHandlers: nil,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// nil handlers should be acceptable
		if llm.config.ToolHandlers != nil && len(llm.config.ToolHandlers) > 0 {
			t.Error("expected ToolHandlers to be nil or empty")
		}
	})

	t.Run("empty handlers map", func(t *testing.T) {
		llm, err := New(Config{
			ToolHandlers: map[string]ToolHandler{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if llm.config.ToolHandlers == nil {
			t.Fatal("expected ToolHandlers to be set")
		}
		if len(llm.config.ToolHandlers) != 0 {
			t.Errorf("expected 0 handlers, got %d", len(llm.config.ToolHandlers))
		}
	})

	t.Run("handler registration persists after New", func(t *testing.T) {
		handler := func(args map[string]any) (string, error) {
			return "persisted", nil
		}

		config := Config{
			ToolHandlers: map[string]ToolHandler{
				"persistent_tool": handler,
			},
		}

		llm, err := New(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify handler is still accessible
		if h, ok := llm.config.ToolHandlers["persistent_tool"]; !ok {
			t.Error("expected persistent_tool handler to be registered")
		} else {
			result, err := h(map[string]any{})
			if err != nil {
				t.Errorf("unexpected error calling handler: %v", err)
			}
			if result != "persisted" {
				t.Errorf("expected 'persisted', got %q", result)
			}
		}
	})
}

func TestToolHandlerExecution(t *testing.T) {
	// Test that handlers are called correctly
	// Note: This is a unit test for the handler function itself
	t.Run("successful handler execution", func(t *testing.T) {
		called := false
		var receivedArgs map[string]any

		handler := func(args map[string]any) (string, error) {
			called = true
			receivedArgs = args
			return "success result", nil
		}

		// Simulate handler invocation
		args := map[string]any{
			"location": "San Francisco",
			"units":    "celsius",
		}
		result, err := handler(args)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected handler to be called")
		}
		if result != "success result" {
			t.Errorf("expected 'success result', got %q", result)
		}
		if receivedArgs["location"] != "San Francisco" {
			t.Errorf("expected location 'San Francisco', got %v", receivedArgs["location"])
		}
		if receivedArgs["units"] != "celsius" {
			t.Errorf("expected units 'celsius', got %v", receivedArgs["units"])
		}
	})

	t.Run("handler with error", func(t *testing.T) {
		handler := func(args map[string]any) (string, error) {
			return "", fmt.Errorf("handler error: invalid location")
		}

		result, err := handler(map[string]any{"location": "invalid"})

		if err == nil {
			t.Error("expected error, got nil")
		}
		if err != nil && !stringContains(err.Error(), "handler error") {
			t.Errorf("expected 'handler error' in error message, got: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty result on error, got %q", result)
		}
	})

	t.Run("handler with empty args", func(t *testing.T) {
		called := false
		handler := func(args map[string]any) (string, error) {
			called = true
			if len(args) != 0 {
				return "", fmt.Errorf("expected empty args, got %d", len(args))
			}
			return "ok", nil
		}

		result, err := handler(map[string]any{})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected handler to be called")
		}
		if result != "ok" {
			t.Errorf("expected 'ok', got %q", result)
		}
	})

	t.Run("handler with nil args", func(t *testing.T) {
		handler := func(args map[string]any) (string, error) {
			if args != nil {
				return "", fmt.Errorf("expected nil args")
			}
			return "nil ok", nil
		}

		result, err := handler(nil)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != "nil ok" {
			t.Errorf("expected 'nil ok', got %q", result)
		}
	})

	t.Run("handler with complex args", func(t *testing.T) {
		var receivedArgs map[string]any

		handler := func(args map[string]any) (string, error) {
			receivedArgs = args
			return "processed", nil
		}

		args := map[string]any{
			"string": "value",
			"number": 42,
			"float":  3.14,
			"bool":   true,
			"array":  []string{"a", "b", "c"},
			"nested": map[string]any{
				"key": "nested value",
			},
		}

		result, err := handler(args)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != "processed" {
			t.Errorf("expected 'processed', got %q", result)
		}
		if receivedArgs["string"] != "value" {
			t.Errorf("expected string 'value', got %v", receivedArgs["string"])
		}
		if receivedArgs["number"] != 42 {
			t.Errorf("expected number 42, got %v", receivedArgs["number"])
		}
		if receivedArgs["bool"] != true {
			t.Errorf("expected bool true, got %v", receivedArgs["bool"])
		}
	})

	t.Run("handler modifying args doesn't affect caller", func(t *testing.T) {
		handler := func(args map[string]any) (string, error) {
			args["modified"] = true
			return "modified", nil
		}

		originalArgs := map[string]any{
			"original": "value",
		}

		_, err := handler(originalArgs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Args were modified by handler (this is expected behavior)
		if originalArgs["modified"] != true {
			t.Error("expected args to be modified")
		}
	})

	t.Run("multiple handler executions are independent", func(t *testing.T) {
		callCount := 0
		handler := func(args map[string]any) (string, error) {
			callCount++
			return fmt.Sprintf("call_%d", callCount), nil
		}

		result1, _ := handler(map[string]any{})
		result2, _ := handler(map[string]any{})
		result3, _ := handler(map[string]any{})

		if callCount != 3 {
			t.Errorf("expected 3 calls, got %d", callCount)
		}
		if result1 != "call_1" || result2 != "call_2" || result3 != "call_3" {
			t.Errorf("unexpected results: %s, %s, %s", result1, result2, result3)
		}
	})
}

func TestMissingToolHandler(t *testing.T) {
	// Test error handling when a tool is defined but no handler is registered
	t.Run("missing handler for single tool", func(t *testing.T) {
		llm, err := New(Config{
			ToolHandlers: map[string]ToolHandler{
				// No handlers registered
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		genaiTools := []*genai.Tool{
			{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{
						Name:        "get_weather",
						Description: "Get weather",
					},
				},
			},
		}

		_, err = llm.convertTools(genaiTools)
		if err == nil {
			t.Fatal("expected error for missing handler, got nil")
		}
		if !stringContains(err.Error(), "no handler") && !stringContains(err.Error(), "get_weather") {
			t.Errorf("expected error about missing handler for get_weather, got: %v", err)
		}
	})

	t.Run("missing handler for one of multiple tools", func(t *testing.T) {
		handler := func(args map[string]any) (string, error) {
			return "result", nil
		}

		llm, err := New(Config{
			ToolHandlers: map[string]ToolHandler{
				"get_weather": handler,
				// missing handler for "calculate"
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		genaiTools := []*genai.Tool{
			{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{Name: "get_weather"},
					{Name: "calculate"}, // Missing handler
				},
			},
		}

		_, err = llm.convertTools(genaiTools)
		if err == nil {
			t.Fatal("expected error for missing handler, got nil")
		}
		if !stringContains(err.Error(), "calculate") {
			t.Errorf("expected 'calculate' in error message, got: %v", err)
		}
	})

	t.Run("nil handler map with tool definitions", func(t *testing.T) {
		llm, err := New(Config{
			ToolHandlers: nil,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		genaiTools := []*genai.Tool{
			{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{Name: "get_weather"},
				},
			},
		}

		_, err = llm.convertTools(genaiTools)
		if err == nil {
			t.Fatal("expected error for missing handler with nil map, got nil")
		}
	})

	t.Run("empty handler map with tool definitions", func(t *testing.T) {
		llm, err := New(Config{
			ToolHandlers: map[string]ToolHandler{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		genaiTools := []*genai.Tool{
			{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{Name: "get_weather"},
				},
			},
		}

		_, err = llm.convertTools(genaiTools)
		if err == nil {
			t.Fatal("expected error for missing handler with empty map, got nil")
		}
	})

	t.Run("error message includes tool name", func(t *testing.T) {
		llm, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		genaiTools := []*genai.Tool{
			{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{Name: "specific_tool_name"},
				},
			},
		}

		_, err = llm.convertTools(genaiTools)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !stringContains(err.Error(), "specific_tool_name") {
			t.Errorf("expected tool name in error message, got: %v", err)
		}
	})
}

// Helper function for string contains check
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
