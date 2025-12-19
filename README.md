# adk-copilot-llm

A Go module that implements the [adk-go](https://github.com/google/adk-go) LLM interface for GitHub Copilot. This allows you to use GitHub Copilot as the underlying LLM in agents built with the ADK (Agent Development Kit).

## Features

- ✅ Implements the `model.LLM` interface from adk-go
- ✅ Multiple authentication methods (PAT and OAuth device flow)
- ✅ GitHub Personal Access Token (PAT) support for direct usage
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

### Option 1: Using GitHub Personal Access Token (Recommended)

GitHub Personal Access Tokens (starting with `github_pat_`) are used directly without token exchange, providing simpler authentication:

```go
package main

import (
    "context"
    "log"

    "github.com/ekroon/adk-copilot-llm/copilot"
    "google.golang.org/adk/model"
    "google.golang.org/genai"
)

func main() {
    ctx := context.Background()

    // Create Copilot LLM instance with PAT token
    // PAT tokens are used directly without exchange
    llm, err := copilot.New(copilot.Config{
        GitHubToken: "github_pat_YOUR_TOKEN_HERE",
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

### Option 2: Using Environment Variable

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
    // Supports both PAT tokens (github_pat_*) and OAuth tokens
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

### Option 3: OAuth Device Flow Authentication

For OAuth-based authentication that exchanges tokens for Copilot API keys:

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
    // GitHubToken is the GitHub token for authentication (required)
    // Supports two types:
    // - Personal Access Token (PAT): Tokens starting with "github_pat_"
    //   are used directly without token exchange
    // - OAuth Access Token: Regular OAuth tokens that will be exchanged
    //   for Copilot API keys through the token exchange API
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

## Authentication Methods

This library supports two authentication methods:

### 1. GitHub Personal Access Token (PAT)

Personal Access Tokens (starting with `github_pat_`) are the simplest authentication method. They are used directly without token exchange:

```go
llm, err := copilot.New(copilot.Config{
    GitHubToken: "github_pat_YOUR_TOKEN_HERE",
    Model:       "gpt-4",
})
```

**Advantages:**
- No token exchange API calls needed
- Simpler setup
- Direct usage

### 2. OAuth Token Exchange

OAuth tokens obtained through the device flow are exchanged for Copilot API keys:

```go
auth := copilot.NewAuthenticator(copilot.AuthConfig{})
token, err := auth.Authenticate(ctx)

llm, err := copilot.New(copilot.Config{
    GitHubToken: token,
    Model:       "gpt-4",
})
```

**Advantages:**
- Standard OAuth flow
- Automatic token refresh

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

```bash
# Run the example
cd examples
GITHUB_TOKEN=your_token_here go run main.go
```

If you don't have a token, the example will guide you through the device flow authentication.

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
