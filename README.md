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
- **GitHub Copilot CLI installed and authenticated**

### Installing the Copilot CLI

The Copilot CLI handles all authentication automatically. Install it following the [official instructions](https://docs.github.com/en/copilot/github-copilot-in-the-cli).

Once installed, authenticate with:

```bash
gh auth login
gh extension install github/gh-copilot
```

You can also set the `COPILOT_CLI_PATH` environment variable if the CLI is installed in a non-standard location.

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
