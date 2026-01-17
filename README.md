# adk-copilot-llm

A Go module that implements the [adk-go](https://github.com/google/adk-go) LLM interface for GitHub Copilot. This allows you to use GitHub Copilot as the underlying LLM in agents built with the ADK (Agent Development Kit).

## Features

- Implements the `model.LLM` interface from adk-go
- Streaming and non-streaming content generation
- Multi-turn conversations
- Simple setup - authentication handled by Copilot CLI
- OpenAI-compatible chat completions API

## Installation

```bash
go get github.com/ekroon/adk-copilot-llm
```

## Prerequisites

- Go 1.24 or later
- A GitHub account with access to Copilot
- Active GitHub Copilot subscription
- **GitHub Copilot CLI installed and authenticated** (see [official instructions](https://docs.github.com/en/copilot/github-copilot-in-the-cli))

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ekroon/adk-copilot-llm/copilot"
    "google.golang.org/adk/model"
    "google.golang.org/genai"
)

func main() {
    ctx := context.Background()

    // Create Copilot LLM instance
    // Authentication is handled automatically by the Copilot CLI
    llm, err := copilot.New(copilot.Config{
        Model: "gpt-4",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer llm.Close()

    // Create a request
    request := &model.LLMRequest{
        Contents: []*genai.Content{
            {
                Role:  "user",
                Parts: []*genai.Part{genai.NewPartFromText("Hello!")},
            },
        },
    }

    // Generate content
    for resp, err := range llm.GenerateContent(ctx, request, false) {
        if err != nil {
            log.Fatal(err)
        }
        if resp.Content != nil {
            for _, part := range resp.Content.Parts {
                fmt.Print(part.Text)
            }
        }
    }
    fmt.Println()
}
```

## Configuration

The `Config` struct supports the following options:

```go
type Config struct {
    // CLIPath is the path to the Copilot CLI executable
    // Default: "copilot" (or COPILOT_CLI_PATH environment variable)
    CLIPath string

    // CLIUrl is the URL of an existing CLI server (optional)
    // If provided, connects to an existing server instead of starting a new one
    CLIUrl string

    // Model is the model identifier to use
    // Default: "gpt-4"
    Model string

    // Streaming enables streaming responses by default
    Streaming bool

    // LogLevel sets the logging verbosity
    // Default: "error"
    LogLevel string
}
```

### Environment Variables

- `COPILOT_CLI_PATH`: Path to the Copilot CLI executable (overrides default)

## Streaming

The implementation supports streaming responses for real-time output:

```go
// Enable streaming with the third parameter
for resp, err := range llm.GenerateContent(ctx, request, true) {
    if err != nil {
        log.Fatal(err)
    }
    if resp.Content != nil {
        for _, part := range resp.Content.Parts {
            fmt.Print(part.Text) // Print as tokens arrive
        }
    }
    if resp.TurnComplete {
        break
    }
}
```

## Multi-turn Conversations

Build conversations with multiple turns:

```go
conversation := &model.LLMRequest{
    Contents: []*genai.Content{
        {
            Role:  "user",
            Parts: []*genai.Part{genai.NewPartFromText("My favorite color is blue")},
        },
        {
            Role:  "model",
            Parts: []*genai.Part{genai.NewPartFromText("That's nice! Blue is a calming color.")},
        },
        {
            Role:  "user",
            Parts: []*genai.Part{genai.NewPartFromText("What was my favorite color?")},
        },
    },
}

for resp, err := range llm.GenerateContent(ctx, conversation, false) {
    // Handle response...
}
```

## Examples

See the [examples](./examples) directory for complete working examples:

```bash
cd examples
go run main.go
```

## Tool Support

Tool support enables the LLM to call external functions during conversation, allowing the model to perform actions like querying databases, calling APIs, or executing calculations. Tools are defined using the `google.golang.org/adk/tool` interface and created with `functiontool.New()` for type-safe tool definitions with automatic JSON schema generation. The copilot SDK automatically handles tool execution: when the model decides to use a tool, your handler is invoked, and the result is sent back to continue the conversation.

### Example

```go
import (
    "github.com/ekroon/adk-copilot-llm/copilot"
    "google.golang.org/adk/tool"
    "google.golang.org/adk/tool/functiontool"
)

// Define input/output types with JSON schema tags
type CalculatorInput struct {
    Operation string  `json:"operation" jsonschema:"enum=add,enum=multiply,enum=divide"`
    A         float64 `json:"a"`
    B         float64 `json:"b"`
}

type CalculatorOutput struct {
    Result float64 `json:"result"`
}

// Create tool using functiontool.New with type-safe handler
calcTool, _ := functiontool.New(functiontool.Config{
    Name:        "calculator",
    Description: "Performs basic arithmetic operations",
}, func(ctx tool.Context, input CalculatorInput) (CalculatorOutput, error) {
    var result float64
    switch input.Operation {
    case "add":
        result = input.A + input.B
    case "multiply":
        result = input.A * input.B
    case "divide":
        if input.B == 0 {
            return CalculatorOutput{}, fmt.Errorf("division by zero")
        }
        result = input.A / input.B
    }
    return CalculatorOutput{Result: result}, nil
})

// Create LLM with tools
llm, _ := copilot.New(copilot.Config{
    Model: "gpt-4",
    Tools: []tool.Tool{calcTool},
})

// Use the LLM - tools are automatically executed
for resp, _ := range llm.GenerateContent(ctx, req, false) {
    // Response includes results from tool executions
}
```

For a complete working example, see [examples/tools/main.go](./examples/tools/main.go).

**Note**: In standalone LLM mode, the `tool.Context` has limited functionality (no session state, memory, or actions). For full adk runtime features, use `llmagent.New()` with your CopilotLLM as the model provider.

## API Compatibility

This library implements the `model.LLM` interface from adk-go:

```go
type LLM interface {
    Name() string
    GenerateContent(ctx context.Context, req *LLMRequest, stream bool) iter.Seq2[*LLMResponse, error]
}
```

Remember to call `Close()` when done to clean up CLI resources:

```go
llm, err := copilot.New(config)
if err != nil {
    log.Fatal(err)
}
defer llm.Close()
```

## License

Apache 2.0 - See LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## References

- [adk-go](https://github.com/google/adk-go) - Agent Development Kit for Go
- [GitHub Copilot CLI](https://docs.github.com/en/copilot/github-copilot-in-the-cli) - CLI installation and setup
- [copilot-sdk](https://github.com/github/copilot-sdk) - Official Copilot SDK
- [GitHub Copilot API](https://docs.github.com/en/copilot)
