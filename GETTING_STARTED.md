# Getting Started with adk-copilot-llm

This guide will help you get started with using GitHub Copilot as your LLM in adk-go agents.

## Prerequisites

Before you begin, ensure you have:

1. **Go 1.24 or later** installed
2. **GitHub account** with an active Copilot subscription
3. **GitHub personal access token** (if using environment variable method) OR ability to complete OAuth device flow

## Installation

Add the package to your Go project:

```bash
go get github.com/ekroon/adk-copilot-llm
```

## Quick Start Guide

### Step 1: Choose Your Authentication Method

You have two options for authentication:

#### Option A: GitHub Personal Access Token (Recommended)

The simplest method is to use a GitHub Personal Access Token (PAT). These tokens start with `github_pat_` and are used directly without token exchange.

**To create a PAT:**
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate a new token with appropriate scopes for Copilot access
3. Copy the token (it starts with `github_pat_`)

```go
package main

import (
    "context"
    "log"
    
    "github.com/ekroon/adk-copilot-llm/copilot"
)

func main() {
    ctx := context.Background()
    
    // Use PAT token directly (no exchange needed)
    llm, err := copilot.New(copilot.Config{
        GitHubToken: "github_pat_YOUR_TOKEN_HERE",
        Model:       "gpt-4",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Use the LLM...
}
```

#### Option B: OAuth Device Flow (For OAuth-based authentication)

This method walks you through the GitHub OAuth process and exchanges the token for Copilot API keys:

```go
package main

import (
    "context"
    "log"
    
    "github.com/ekroon/adk-copilot-llm/copilot"
)

func main() {
    ctx := context.Background()
    
    // Start device flow
    auth := copilot.NewAuthenticator(copilot.AuthConfig{})
    token, err := auth.Authenticate(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Save this token for future use
    log.Printf("Token: %s", token)
}
```

#### Option C: Environment Variable (For production use)

Set your GitHub token as an environment variable (works with both PAT and OAuth tokens):

```bash
export GITHUB_TOKEN=your_token_here
```

Then use it in your code:

```go
llm, err := copilot.New(copilot.Config{
    GitHubToken: os.Getenv("GITHUB_TOKEN"),
})
```

### Step 2: Create Your First Agent

Here's a simple example that uses Copilot to answer questions:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/ekroon/adk-copilot-llm/copilot"
    "google.golang.org/adk/model"
    "google.golang.org/genai"
)

func main() {
    ctx := context.Background()
    
    // Create LLM instance
    llm, err := copilot.New(copilot.Config{
        GitHubToken: os.Getenv("GITHUB_TOKEN"),
        Model:       "gpt-4",
    })
    if err != nil {
        log.Fatal(err)
    }
    
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

### Step 3: Add Streaming for Real-time Responses

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

### Step 4: Build a Multi-turn Conversation

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

## Advanced Configuration

### Temperature and Other Parameters

Control the creativity and randomness of responses:

```go
temp := float32(0.7)
topP := float32(0.9)
maxTokens := int32(500)

req := &model.LLMRequest{
    Contents: contents,
    Config: &genai.GenerateContentConfig{
        Temperature:     &temp,
        TopP:            &topP,
        MaxOutputTokens: maxTokens,
    },
}
```

### GitHub Enterprise

For GitHub Enterprise deployments:

```go
// For authentication
auth := copilot.NewAuthenticator(copilot.AuthConfig{
    EnterpriseURL: "company.ghe.com",
})

// For LLM instance
llm, err := copilot.New(copilot.Config{
    GitHubToken:   token,
    EnterpriseURL: "company.ghe.com",
    Model:         "gpt-4",
})
```

## Troubleshooting

### Authentication Issues

**Problem**: "failed to fetch API key: status 401"
**Solution**: Your GitHub token may have expired. Re-authenticate using the device flow.

**Problem**: "GitHubToken is required"
**Solution**: Make sure you've set the `GITHUB_TOKEN` environment variable or provided it in the config.

### Rate Limiting

GitHub Copilot has rate limits. If you encounter rate limit errors:
- Implement exponential backoff
- Cache responses when possible
- Use appropriate temperature settings to get better responses on the first try

### Enterprise Configuration

**Problem**: Connection timeout with enterprise URL
**Solution**: Ensure your enterprise URL is accessible and that you've provided it in both the authenticator and LLM config.

## Best Practices

1. **Token Management**: Store tokens securely, never commit them to version control
2. **Error Handling**: Always check for errors in the response iterator
3. **Streaming**: Use streaming for better user experience in interactive applications
4. **Context Management**: Pass proper context for cancellation support
5. **Model Selection**: Use `gpt-4` for complex tasks, `gpt-3.5-turbo` for faster, simpler responses

## Next Steps

- Check out the [examples](./examples) directory for more use cases
- Read the [API documentation](./README.md) for detailed reference
- Explore the [adk-go documentation](https://github.com/google/adk-go) for building complete agents

## Support

For issues and questions:
- Open an issue on [GitHub](https://github.com/ekroon/adk-copilot-llm/issues)
- Check existing issues for solutions
- Refer to the [adk-go documentation](https://github.com/google/adk-go)
