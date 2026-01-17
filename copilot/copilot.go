// Package copilot provides an implementation of the adk-go LLM interface
// for GitHub Copilot, using the official copilot-sdk.
package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"strings"
	"sync"
	"time"

	copilot "github.com/github/copilot-sdk/go"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// Config holds the configuration for the Copilot LLM.
type Config struct {
	// CLIPath is the path to the Copilot CLI executable (default: "copilot" or COPILOT_CLI_PATH env)
	CLIPath string
	// CLIUrl is the URL of an existing CLI server (optional, e.g., "localhost:8080")
	CLIUrl string
	// Model is the model identifier (default: "gpt-4")
	Model string
	// Streaming enables streaming responses by default
	Streaming bool
	// LogLevel for the copilot client (default: "error")
	LogLevel string
	// Tools is a list of tools available to the LLM.
	// Each tool must implement google.golang.org/adk/tool.Tool and provide
	// a Declaration() method for schema and Run() method for execution.
	Tools []tool.Tool
}

// CopilotLLM implements the model.LLM interface for GitHub Copilot.
type CopilotLLM struct {
	config  Config
	client  *copilot.Client
	started bool
	mu      sync.Mutex
}

// toolContext provides a minimal implementation of tool.Context for copilot-based tool execution.
// This is a simplified context that doesn't have full adk agent runtime features.
// For full context support (session state, memory, actions), use llmagent.New() with adk's agent runtime.
type toolContext struct {
	ctx    context.Context
	callID string
}

func (tc *toolContext) Context() context.Context {
	return tc.ctx
}

func (tc *toolContext) FunctionCallID() string {
	return tc.callID
}

// Agent returns nil as we don't have an agent runtime in standalone LLM mode.
func (tc *toolContext) Agent() agent.Agent {
	return nil
}

// Session returns nil as we don't have session management in standalone LLM mode.
func (tc *toolContext) Session() *session.Session {
	return nil
}

// Actions returns nil as event actions are not available in standalone LLM mode.
func (tc *toolContext) Actions() *session.EventActions {
	return nil
}

// SearchMemory returns an error as memory search is not available in standalone LLM mode.
func (tc *toolContext) SearchMemory(ctx context.Context, query string) (*memory.SearchResponse, error) {
	return nil, fmt.Errorf("memory search not available in standalone LLM mode; use llmagent.New() for full adk runtime support")
}

// AgentName returns empty string as we don't have agent runtime in standalone LLM mode.
func (tc *toolContext) AgentName() string {
	return ""
}

// AppName returns empty string as we don't have app context in standalone LLM mode.
func (tc *toolContext) AppName() string {
	return ""
}

// Artifacts returns nil as artifacts are not available in standalone LLM mode.
func (tc *toolContext) Artifacts() agent.Artifacts {
	return nil
}

// context.Context interface methods
func (tc *toolContext) Deadline() (deadline time.Time, ok bool) {
	return tc.ctx.Deadline()
}

func (tc *toolContext) Done() <-chan struct{} {
	return tc.ctx.Done()
}

func (tc *toolContext) Err() error {
	return tc.ctx.Err()
}

func (tc *toolContext) Value(key interface{}) interface{} {
	return tc.ctx.Value(key)
}

// ReadonlyContext interface methods
func (tc *toolContext) UserContent() *genai.Content {
	return nil
}

func (tc *toolContext) InvocationID() string {
	return ""
}

func (tc *toolContext) ReadonlyState() session.ReadonlyState {
	return nil
}

func (tc *toolContext) UserID() string {
	return ""
}

func (tc *toolContext) SessionID() string {
	return ""
}

func (tc *toolContext) Branch() string {
	return ""
}

func (tc *toolContext) State() session.State {
	return nil
}

// New creates a new CopilotLLM instance with the given configuration.
func New(cfg Config) (*CopilotLLM, error) {
	// Apply defaults
	if cfg.Model == "" {
		cfg.Model = "gpt-4"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}
	if cfg.CLIPath == "" {
		if envPath := os.Getenv("COPILOT_CLI_PATH"); envPath != "" {
			cfg.CLIPath = envPath
		} else {
			cfg.CLIPath = "copilot"
		}
	}

	// Create client options
	opts := &copilot.ClientOptions{
		CLIPath:  cfg.CLIPath,
		LogLevel: cfg.LogLevel,
	}
	if cfg.CLIUrl != "" {
		opts.CLIUrl = cfg.CLIUrl
	}

	// Create the client (but don't start it yet - lazy start in GenerateContent)
	client := copilot.NewClient(opts)

	return &CopilotLLM{
		config:  cfg,
		client:  client,
		started: false,
	}, nil
}

// Name returns the name of this LLM implementation.
func (c *CopilotLLM) Name() string {
	return "github-copilot"
}

// Close stops the copilot client gracefully.
func (c *CopilotLLM) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started && c.client != nil {
		c.client.Stop()
		c.started = false
	}
	return nil
}

// ensureStarted ensures the client is started (lazy initialization).
func (c *CopilotLLM) ensureStarted() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}

	if err := c.client.Start(); err != nil {
		return fmt.Errorf("failed to start copilot client: %w", err)
	}
	c.started = true
	return nil
}

// GenerateContent implements the model.LLM interface's GenerateContent method.
func (c *CopilotLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Ensure client is started (lazy start)
		if err := c.ensureStarted(); err != nil {
			yield(nil, fmt.Errorf("failed to start client: %w", err))
			return
		}

		// Determine model to use
		modelName := c.config.Model
		if req.Model != "" {
			modelName = req.Model
		}

		// Determine streaming mode
		streaming := c.config.Streaming
		if stream {
			streaming = true
		}

		// Convert adk tools to copilot tools
		var copilotTools []copilot.Tool
		if len(c.config.Tools) > 0 {
			var err error
			copilotTools, err = c.convertAdkTools(ctx, c.config.Tools)
			if err != nil {
				yield(nil, fmt.Errorf("failed to convert tools: %w", err))
				return
			}
		}

		// Create a new session for this request
		session, err := c.client.CreateSession(&copilot.SessionConfig{
			Model:     modelName,
			Streaming: streaming,
			Tools:     copilotTools,
		})
		if err != nil {
			yield(nil, fmt.Errorf("failed to create session: %w", err))
			return
		}
		defer session.Destroy()

		// Format the prompt from the request contents
		prompt := formatPrompt(req.Contents)

		// Create channels to bridge event callbacks to iterator
		// Use larger buffer to prevent blocking in the event callback goroutine
		type eventResult struct {
			response *model.LLMResponse
			err      error
			done     bool
		}
		eventCh := make(chan eventResult, 100)

		// Track if we've already received the final message to avoid duplicate TurnComplete
		var receivedFinalMessage bool

		// Subscribe to session events
		unsubscribe := session.On(func(event copilot.SessionEvent) {
			switch event.Type {
			case "assistant.message_delta":
				// Streaming partial response
				if streaming && event.Data.DeltaContent != nil {
					resp := convertEventToResponse(event, true)
					select {
					case eventCh <- eventResult{response: resp}:
					default:
						// Drop if channel is full to prevent blocking
					}
				}
			case "assistant.message":
				// Final complete message
				receivedFinalMessage = true
				resp := convertEventToResponse(event, false)
				select {
				case eventCh <- eventResult{response: resp}:
				default:
					// Drop if channel is full to prevent blocking
				}
			case "session.idle":
				// Turn is complete - only send if we haven't sent a final message
				// (the final message already has TurnComplete: true)
				if !receivedFinalMessage {
					select {
					case eventCh <- eventResult{done: true}:
					default:
					}
				} else {
					// Signal done without sending another TurnComplete response
					select {
					case eventCh <- eventResult{done: true}:
					default:
					}
				}
			case "session.error":
				// Handle error events from the SDK
				errMsg := "unknown error"
				if event.Data.Content != nil {
					errMsg = *event.Data.Content
				}
				select {
				case eventCh <- eventResult{err: fmt.Errorf("session error: %s", errMsg)}:
				default:
				}
			}
		})
		defer unsubscribe()

		// Send the message
		_, err = session.Send(copilot.MessageOptions{
			Prompt: prompt,
		})
		if err != nil {
			yield(nil, fmt.Errorf("failed to send message: %w", err))
			return
		}

		// Process events from the channel
		for {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			case result := <-eventCh:
				if result.err != nil {
					yield(nil, result.err)
					return
				}
				if result.done {
					// Done signal - just return, don't send another TurnComplete
					// since the final assistant.message already has TurnComplete: true
					return
				}
				if result.response != nil {
					if !yield(result.response, nil) {
						return
					}
				}
			}
		}
	}
}

// formatPrompt converts the conversation history to a prompt string.
func formatPrompt(contents []*genai.Content) string {
	if len(contents) == 0 {
		return ""
	}

	// If there's only one content, just extract its text
	if len(contents) == 1 {
		return extractText(contents[0])
	}

	// Format multi-turn conversation
	var sb strings.Builder
	for _, content := range contents {
		role := strings.ToLower(content.Role)
		text := extractText(content)

		if text == "" {
			continue
		}

		// Format as conversation
		switch role {
		case "user":
			sb.WriteString("User: ")
		case "model", "assistant":
			sb.WriteString("Assistant: ")
		case "system":
			sb.WriteString("System: ")
		default:
			sb.WriteString(role)
			sb.WriteString(": ")
		}
		sb.WriteString(text)
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String())
}

// extractText extracts text content from a genai.Content.
func extractText(content *genai.Content) string {
	if content == nil || len(content.Parts) == 0 {
		return ""
	}

	var texts []string
	for _, part := range content.Parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}

	return strings.Join(texts, "\n")
}

// convertEventToResponse converts a copilot session event to an LLMResponse.
func convertEventToResponse(event copilot.SessionEvent, partial bool) *model.LLMResponse {
	resp := &model.LLMResponse{
		Partial:      partial,
		TurnComplete: !partial,
	}

	var text string
	if partial && event.Data.DeltaContent != nil {
		text = *event.Data.DeltaContent
	} else if !partial && event.Data.Content != nil {
		text = *event.Data.Content
		resp.FinishReason = genai.FinishReasonStop
	}

	if text != "" {
		resp.Content = &genai.Content{
			Role:  "model",
			Parts: []*genai.Part{genai.NewPartFromText(text)},
		}
	}

	return resp
}

// schemaToMap converts a genai.Schema to a map[string]interface{} for copilot tools.
func schemaToMap(schema *genai.Schema) map[string]interface{} {
	if schema == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Type
	if schema.Type != "" {
		result["type"] = strings.ToLower(string(schema.Type))
	}

	// Description
	if schema.Description != "" {
		result["description"] = schema.Description
	}

	// Properties (for object type)
	if len(schema.Properties) > 0 {
		props := make(map[string]interface{})
		for name, propSchema := range schema.Properties {
			props[name] = schemaToMap(propSchema)
		}
		result["properties"] = props
	}

	// Required fields
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	// Items (for array type)
	if schema.Items != nil {
		result["items"] = schemaToMap(schema.Items)
	}

	// Enum
	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}

	// Format
	if schema.Format != "" {
		result["format"] = schema.Format
	}

	// Numeric constraints
	if schema.Minimum != nil {
		result["minimum"] = *schema.Minimum
	}
	if schema.Maximum != nil {
		result["maximum"] = *schema.Maximum
	}

	// String constraints
	if schema.MinLength != nil {
		result["minLength"] = *schema.MinLength
	}
	if schema.MaxLength != nil {
		result["maxLength"] = *schema.MaxLength
	}
	if schema.Pattern != "" {
		result["pattern"] = schema.Pattern
	}

	// Array constraints
	if schema.MinItems != nil {
		result["minItems"] = *schema.MinItems
	}
	if schema.MaxItems != nil {
		result["maxItems"] = *schema.MaxItems
	}

	// Object constraints
	if schema.MinProperties != nil {
		result["minProperties"] = *schema.MinProperties
	}
	if schema.MaxProperties != nil {
		result["maxProperties"] = *schema.MaxProperties
	}

	// Nullable
	if schema.Nullable != nil {
		result["nullable"] = *schema.Nullable
	}

	// Default value
	if schema.Default != nil {
		result["default"] = schema.Default
	}

	// AnyOf
	if len(schema.AnyOf) > 0 {
		anyOf := make([]interface{}, len(schema.AnyOf))
		for i, s := range schema.AnyOf {
			anyOf[i] = schemaToMap(s)
		}
		result["anyOf"] = anyOf
	}

	return result
}

// convertAdkTools converts adk tool.Tool instances to copilot.Tool instances.
func (c *CopilotLLM) convertAdkTools(ctx context.Context, tools []tool.Tool) ([]copilot.Tool, error) {
	copilotTools := make([]copilot.Tool, 0, len(tools))

	for _, t := range tools {
		// Check if the tool implements the FunctionTool interface (Declaration and Run methods)
		// We use interface assertion instead of importing internal package
		type functionTool interface {
			tool.Tool
			Declaration() *genai.FunctionDeclaration
			Run(tool.Context, any) (map[string]any, error)
		}

		ft, ok := t.(functionTool)
		if !ok {
			return nil, fmt.Errorf("tool %q does not implement required methods (Declaration and Run)", t.Name())
		}

		decl := ft.Declaration()
		if decl == nil {
			return nil, fmt.Errorf("tool %q returned nil declaration", t.Name())
		}

		// Convert declaration parameters to copilot format
		params := declarationToParams(decl)

		// Create copilot tool with wrapper handler that calls the adk tool's Run method
		// Use closure variable to avoid capturing loop variable
		toolRef := ft
		toolName := t.Name()

		copilotTools = append(copilotTools, copilot.Tool{
			Name:        toolName,
			Description: decl.Description,
			Parameters:  params,
			Handler: func(inv copilot.ToolInvocation) (copilot.ToolResult, error) {
				// Create minimal tool context
				tc := &toolContext{
					ctx:    ctx,
					callID: inv.ToolCallID,
				}

				// Call the adk tool's Run method
				result, err := toolRef.Run(tc, inv.Arguments)
				if err != nil {
					return copilot.ToolResult{
						Error: err.Error(),
					}, nil
				}

				// Convert result to JSON string for LLM
				resultJSON, err := json.Marshal(result)
				if err != nil {
					return copilot.ToolResult{
						Error: fmt.Sprintf("failed to marshal result: %v", err),
					}, nil
				}

				return copilot.ToolResult{
					TextResultForLLM: string(resultJSON),
				}, nil
			},
		})
	}

	return copilotTools, nil
}

// declarationToParams converts a genai.FunctionDeclaration's parameters to copilot tool parameter format.
func declarationToParams(decl *genai.FunctionDeclaration) map[string]interface{} {
	if decl.ParametersJsonSchema != nil {
		// If ParametersJsonSchema is provided, use it directly
		if m, ok := decl.ParametersJsonSchema.(map[string]interface{}); ok {
			return m
		}
	}

	if decl.Parameters != nil {
		// Convert genai.Schema to map[string]interface{}
		return schemaToMap(decl.Parameters)
	}

	return nil
}
