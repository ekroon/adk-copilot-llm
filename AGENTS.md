# Agent Guidelines for adk-copilot-llm

## Overview

adk-copilot-llm provides a GitHub Copilot implementation of the `model.LLM` interface from [google.golang.org/adk](https://pkg.go.dev/google.golang.org/adk). This allows developers to use GitHub Copilot models (GPT-4, Claude, etc.) within the ADK ecosystem.

**Key Architecture Points:**
- Implements `model.LLM` interface from adk-go v0.3.0+
- Uses official [copilot-sdk](https://github.com/github/copilot-sdk) for LLM communication
- Supports tool execution via `google.golang.org/adk/tool` interface
- Can be used standalone or with full ADK agent runtime (`llmagent.New()`)

## Build/Test Commands

### Development
- Run all tests: `go test -v ./...` or `make test`
- Run single test: `go test -v ./copilot -run TestName`
- Run vet (linting): `go vet ./...` or `make vet`
- Install deps: `go mod download` or `make setup`
- Build copilot package: `go build ./copilot`

### Examples
- Run basic example: `cd examples && GITHUB_TOKEN=your_token go run main.go`
- Run tools example: `cd examples/tools && GITHUB_TOKEN=your_token go run main.go`
- Build example binary: `cd examples/tools && go build .`

**Note**: The `GITHUB_TOKEN` environment variable must be set with a GitHub Personal Access Token that has Copilot access.

## Code Style

- **Language**: Go 1.24+
- **Imports**: Standard library first, external packages second, blank line between groups
- **Formatting**: Use `gofmt` (tabs for indentation)
- **Types**: Exported types have package doc comments. Struct fields documented inline when needed
- **Naming**: Use camelCase for unexported, PascalCase for exported. Acronyms stay uppercase (e.g., `HTTPClient`, `apiKeyURL`)
- **Error handling**: Always wrap errors with context using `fmt.Errorf("description: %w", err)`
- **Constants**: Group related constants together, use `const` blocks with comments
- **Interfaces**: Follow standard Go interface patterns (e.g., `model.LLM` from adk-go)
- **Concurrency**: Use `sync.RWMutex` for thread-safe shared state, prefer read locks when possible
- **Testing**: Use table-driven tests with `t.Run()` for subtests. Test both success and error cases

## Dependencies

### Core Dependencies
- `google.golang.org/adk` v0.3.0+ - ADK Go framework
- `google.golang.org/genai` v1.40.0+ - GenAI types and interfaces
- `github.com/github/copilot-sdk/go` - Official GitHub Copilot SDK

### Why These Versions?
- **adk v0.3.0+**: Includes `tool.Tool` interface and `functiontool` package for type-safe tool definitions
- **genai v1.40.0+**: Transitive dependency of adk v0.3.0, provides updated GenAI types
- **copilot-sdk**: Latest version for GitHub Copilot integration

## Tool Support Architecture

### How Tools Work

Tools in adk-copilot-llm follow this flow:

```
User provides tool.Tool → Extract Declaration() for schema → 
Convert to copilot.Tool → Register with copilot SDK →
LLM decides to call tool → copilot CLI requests execution →
SDK calls our handler → We call tool.Run() → Return result →
LLM continues with result → User gets final response
```

### Tool Interface Requirements

Tools must implement the `tool.Tool` interface from `google.golang.org/adk/tool`:

```go
type Tool interface {
    Name() string
    Description() string
    IsLongRunning() bool
}
```

Additionally, for execution support, tools must implement:

```go
Declaration() *genai.FunctionDeclaration
Run(tool.Context, any) (map[string]any, error)
```

This is automatically satisfied by tools created with `functiontool.New()`.

### Tool Context Limitations

The `toolContext` implementation in standalone LLM mode provides:
- ✅ `Context()` - Standard Go context
- ✅ `FunctionCallID()` - Tool invocation ID
- ❌ `Agent()` - Returns nil (no agent runtime)
- ❌ `Session()` - Returns nil (no session management)
- ❌ `Actions()` - Returns nil (no event actions)
- ❌ `SearchMemory()` - Returns error (no memory)

**For full context features**, use `llmagent.New()` with CopilotLLM as the model provider.

### Creating Tools

**Recommended approach** using `functiontool.New()`:

```go
import "google.golang.org/adk/tool/functiontool"

type CalculatorInput struct {
    Operation string  `json:"operation" jsonschema:"enum=add,enum=multiply"`
    A         float64 `json:"a"`
    B         float64 `json:"b"`
}

type CalculatorOutput struct {
    Result float64 `json:"result"`
}

calcTool, err := functiontool.New(functiontool.Config{
    Name:        "calculator",
    Description: "Performs arithmetic operations",
}, func(ctx tool.Context, input CalculatorInput) (CalculatorOutput, error) {
    var result float64
    switch input.Operation {
    case "add":
        result = input.A + input.B
    case "multiply":
        result = input.A * input.B
    }
    return CalculatorOutput{Result: result}, nil
})
```

Benefits:
- Type-safe input/output with Go structs
- Automatic JSON schema generation from struct tags
- Built-in validation via `jsonschema` tags
- Clean error handling

### Tool Conversion Process

When you pass tools to `Config.Tools`, the implementation:

1. **Validates** each tool implements required interfaces
2. **Extracts** `Declaration()` for the function schema
3. **Converts** genai.FunctionDeclaration to copilot.Tool format using `declarationToParams()`
4. **Creates** wrapper handler that:
   - Instantiates minimal `toolContext` with invocation context
   - Calls the tool's `Run(ctx, args)` method
   - Marshals result to JSON for LLM
   - Handles errors and returns `copilot.ToolResult`

### Testing Tools

When writing tests for tool functionality:

```go
func TestMyTool(t *testing.T) {
    // Create tool
    tool, err := functiontool.New(...)
    
    // Create LLM with tool
    llm, err := copilot.New(copilot.Config{
        Model: "gpt-4",
        Tools: []tool.Tool{tool},
    })
    
    // Test tool execution via LLM
    req := &model.LLMRequest{
        Contents: []*genai.Content{{
            Role:  "user",
            Parts: []*genai.Part{genai.NewPartFromText("Calculate 2 + 2")},
        }},
    }
    
    for resp, err := range llm.GenerateContent(ctx, req, false) {
        // Verify response includes tool result
    }
}
```

## Common Patterns

### Standalone LLM Usage

Simple request/response with automatic tool execution:

```go
llm, _ := copilot.New(copilot.Config{
    Model: "gpt-4",
    Tools: []tool.Tool{myTool},
})

for resp, err := range llm.GenerateContent(ctx, req, false) {
    // Response includes tool execution results
}
```

### Full ADK Agent Integration

For session management, memory, and full context:

```go
import "google.golang.org/adk/agent/llmagent"

// Create LLM (no tools here)
llm, _ := copilot.New(copilot.Config{Model: "gpt-4"})

// Create agent with tools
agent, _ := llmagent.New(llmagent.Config{
    Name:  "my_agent",
    Model: llm,
    Tools: []tool.Tool{myTool},
})

// Run with full ADK runtime
runner := agent.NewRunner(...)
for event := range runner.Run(ctx, ...) {
    // Full event stream with tool execution
}
```

### Multi-turn Conversations

```go
conversation := []*genai.Content{
    {Role: "user", Parts: []*genai.Part{genai.NewPartFromText("Hello")}},
    {Role: "model", Parts: []*genai.Part{genai.NewPartFromText("Hi there!")}},
    {Role: "user", Parts: []*genai.Part{genai.NewPartFromText("What's 2+2?")}},
}

req := &model.LLMRequest{Contents: conversation}
for resp, err := range llm.GenerateContent(ctx, req, false) {
    // Response maintains conversation context
}
```

## Error Handling

### Tool Errors

Tool handlers should return descriptive errors:

```go
func(ctx tool.Context, input MyInput) (MyOutput, error) {
    if input.Value < 0 {
        return MyOutput{}, fmt.Errorf("value must be non-negative, got %d", input.Value)
    }
    // ... tool logic
}
```

The copilot SDK will pass the error back to the LLM, which can retry or explain the error to the user.

### LLM Errors

Check for errors in the response iterator:

```go
for resp, err := range llm.GenerateContent(ctx, req, false) {
    if err != nil {
        return fmt.Errorf("generation failed: %w", err)
    }
    // Process response
}
```

## .gitignore Patterns

Add these to `.gitignore`:

```gitignore
# Build artifacts
examples/tools/tools
examples/main

# IDE
.vscode/
.idea/
*.swp
```

## Breaking Changes

### v0.2.0 → Current (Tool Refactoring)

**Old approach** (now removed):
```go
copilot.New(copilot.Config{
    ToolHandlers: map[string]copilot.ToolHandler{
        "calc": func(args map[string]any) (string, error) { ... },
    },
})
```

**New approach**:
```go
tool, _ := functiontool.New(...)
copilot.New(copilot.Config{
    Tools: []tool.Tool{tool},
})
```

Benefits:
- Type-safe tool definitions
- Automatic schema generation
- Standard ADK integration
- Better testing support
