# adk-copilot-llm

A Go module that implements the [adk-go](https://github.com/google/adk-go) LLM interface for GitHub Copilot. This allows you to use GitHub Copilot as the underlying LLM in agents built with the ADK (Agent Development Kit).

## Features

- ✅ Implements the `model.LLM` interface from adk-go
- ✅ GitHub Copilot authentication via OAuth device flow
- ✅ Token management with automatic refresh
- ✅ Streaming and non-streaming content generation
- ✅ Support for GitHub Enterprise
- ✅ Multi-turn conversations
- ✅ OpenAI-compatible chat completions API

## Installation

```bash
go get github.com/ekroon/adk-copilot-llm
```

## Prerequisites

- Go 1.24 or later
- A GitHub account with access to Copilot
- Active GitHub Copilot subscription

## Quick Start

### Option 1: Using Environment Variable

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/ekroon/adk-copilot-llm/copilot"
    "google.golang.org/adk/model"
    "google.golang.org/genai"
)

func main() {
    ctx := context.Background()

    // Create Copilot LLM instance with token from environment
    llm, err := copilot.New(copilot.Config{
        GitHubToken: os.Getenv("GITHUB_TOKEN"),
        Model:       "gpt-4",
    })
    if err != nil {
        log.Fatal(err)
    }

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
                log.Printf("%s", part.Text)
            }
        }
    }
}
```

### Option 2: Device Flow Authentication

```go
package main

import (
    "context"
    "log"

    "github.com/ekroon/adk-copilot-llm/copilot"
)

func main() {
    ctx := context.Background()

    // Start device flow authentication
    auth := copilot.NewAuthenticator(copilot.AuthConfig{})
    token, err := auth.Authenticate(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Create Copilot LLM instance
    llm, err := copilot.New(copilot.Config{
        GitHubToken: token,
        Model:       "gpt-4",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Use the LLM...
}
```

## Configuration

The `Config` struct supports the following options:

```go
type Config struct {
    // GitHubToken is the GitHub OAuth access token (required)
    GitHubToken string
    
    // EnterpriseURL is the optional GitHub Enterprise URL
    // Example: "company.ghe.com" or "https://company.ghe.com"
    EnterpriseURL string
    
    // Model is the model identifier to use (default: "gpt-4")
    // Options: "gpt-4", "gpt-3.5-turbo"
    Model string
    
    // HTTPClient is an optional custom HTTP client
    HTTPClient *http.Client
}
```

## GitHub Enterprise Support

To use with GitHub Enterprise:

```go
llm, err := copilot.New(copilot.Config{
    GitHubToken:   token,
    EnterpriseURL: "company.ghe.com",
    Model:         "gpt-4",
})
```

For authentication with GitHub Enterprise:

```go
auth := copilot.NewAuthenticator(copilot.AuthConfig{
    EnterpriseURL: "company.ghe.com",
})
token, err := auth.Authenticate(ctx)
```

## Streaming

The implementation supports streaming responses:

```go
// Enable streaming with the third parameter
for resp, err := range llm.GenerateContent(ctx, request, true) {
    if err != nil {
        log.Fatal(err)
    }
    if resp.Content != nil {
        for _, part := range resp.Content.Parts {
            fmt.Print(part.Text)
        }
    }
    if resp.TurnComplete {
        break
    }
}
```

## Examples

See the [examples](./examples) directory for complete working examples:

### Basic Example

```bash
# Run the basic example
cd examples
GITHUB_TOKEN=your_token_here go run main.go
```

If you don't have a token, the example will guide you through the device flow authentication.

### Token Storage with go-keyring

The [with_keyring](./examples/with_keyring) example demonstrates how to securely store and retrieve GitHub tokens using your system's keyring:

```bash
# Run the keyring example
cd examples/with_keyring
go run main.go
```

This example:
- Securely stores tokens in your system's credential manager (Keychain on macOS, Credential Manager on Windows, Secret Service on Linux)
- Automatically retrieves stored tokens for subsequent runs
- Only prompts for authentication when no token is found

See the [keyring example README](./examples/with_keyring/README.md) for more details.

## Authentication Flow

The authentication flow is based on OAuth device flow:

1. **Request Device Code**: The library requests a device code from GitHub
2. **User Authorization**: User visits the verification URL and enters the user code
3. **Token Polling**: The library polls for the access token
4. **Token Refresh**: Copilot API tokens are automatically refreshed as needed

## API Compatibility

This library implements the `model.LLM` interface from adk-go:

```go
type LLM interface {
    Name() string
    GenerateContent(ctx context.Context, req *LLMRequest, stream bool) iter.Seq2[*LLMResponse, error]
}
```

## License

Apache 2.0 - See LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## References

- [adk-go](https://github.com/google/adk-go) - Agent Development Kit for Go
- [GitHub Copilot API](https://docs.github.com/en/copilot)
- [OpenCode Copilot Auth Example](https://github.com/sst/opencode-copilot-auth)
