# Getting Started with adk-copilot-llm

This guide will help you get started with using GitHub Copilot as your LLM in adk-go agents.

## Prerequisites

Before you begin, ensure you have:

1. **Go 1.24 or later** installed
2. **GitHub account** with an active Copilot subscription
3. **GitHub Copilot CLI** installed and authenticated

### Installing the Copilot CLI

The library uses the official Copilot CLI which handles all authentication automatically.

**Step 1: Install GitHub CLI**

```bash
# macOS
brew install gh

# Windows
winget install --id GitHub.cli

# Linux (Debian/Ubuntu)
sudo apt install gh
```

**Step 2: Authenticate with GitHub**

```bash
gh auth login
```

**Step 3: Install Copilot Extension**

```bash
gh extension install github/gh-copilot
```

**Step 4: Verify Installation**

```bash
gh copilot --version
```

## Installation

Add the package to your Go project:

```bash
go get github.com/ekroon/adk-copilot-llm
```

## Quick Start

### Create Your First Agent

Here's a simple example that uses Copilot to answer questions:

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

    // Create LLM instance
    // The CLI handles authentication automatically
    llm, err := copilot.New(copilot.Config{
        Model: "gpt-4",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer llm.Close()

    // Create a simple request
    req := &model.LLMRequest{
        Contents: []*genai.Content{
            {
                Role:  "user",
                Parts: []*genai.Part{
                    genai.NewPartFromText("Explain what an LLM is in simple terms"),
                },
            },
        },
    }

    // Generate response
    for resp, err := range llm.GenerateContent(ctx, req, false) {
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

### Add Streaming for Real-time Responses

For a better user experience, enable streaming:

```go
// Enable streaming with true parameter
for resp, err := range llm.GenerateContent(ctx, req, true) {
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

### Build a Multi-turn Conversation

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

## Configuration Options

```go
llm, err := copilot.New(copilot.Config{
    // Path to CLI executable (optional)
    // Default: "copilot" or COPILOT_CLI_PATH env var
    CLIPath: "/custom/path/to/copilot",

    // Connect to existing CLI server (optional)
    CLIUrl: "http://localhost:8080",

    // Model to use
    Model: "gpt-4",

    // Enable streaming by default
    Streaming: true,

    // Log level: "debug", "info", "warn", "error"
    LogLevel: "error",
})
```

## Troubleshooting

### CLI Not Found

**Problem**: "copilot: command not found" or similar error

**Solution**: 
1. Ensure the Copilot CLI is installed: `gh copilot --version`
2. If installed in a custom location, set the `COPILOT_CLI_PATH` environment variable:
   ```bash
   export COPILOT_CLI_PATH=/path/to/copilot
   ```

### Authentication Issues

**Problem**: Authentication errors when connecting

**Solution**: 
1. Re-authenticate with GitHub CLI:
   ```bash
   gh auth login
   ```
2. Ensure your Copilot subscription is active
3. Check that the Copilot extension is installed:
   ```bash
   gh extension list
   ```

### Connection Timeout

**Problem**: Timeout when starting or connecting to CLI

**Solution**:
1. Check your network connection
2. If using `CLIUrl`, ensure the server is running and accessible
3. Try increasing the timeout or restarting the CLI

### Rate Limiting

GitHub Copilot has rate limits. If you encounter rate limit errors:
- Implement exponential backoff in your application
- Cache responses when possible
- Reduce request frequency

## Best Practices

1. **Always Close**: Call `llm.Close()` when done to clean up CLI resources
2. **Error Handling**: Always check for errors in the response iterator
3. **Streaming**: Use streaming for better user experience in interactive applications
4. **Context Management**: Pass proper context for cancellation support
5. **Resource Cleanup**: Use `defer llm.Close()` immediately after creating the LLM

## Next Steps

- Check out the [examples](./examples) directory for more use cases
- Read the [API documentation](./README.md) for detailed reference
- Explore the [adk-go documentation](https://github.com/google/adk-go) for building complete agents

## Support

For issues and questions:
- Open an issue on [GitHub](https://github.com/ekroon/adk-copilot-llm/issues)
- Check existing issues for solutions
- Refer to the [adk-go documentation](https://github.com/google/adk-go)
