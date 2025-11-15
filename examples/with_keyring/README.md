# GitHub Token Storage with go-keyring

This example demonstrates how to use [github.com/zalando/go-keyring](https://github.com/zalando/go-keyring) to securely store and retrieve GitHub tokens in your system's keyring.

## Features

- Securely stores GitHub tokens in the system keyring (Keychain on macOS, Credential Manager on Windows, Secret Service on Linux)
- Automatically retrieves stored tokens for subsequent runs
- Falls back to device flow authentication if no token is found
- Provides helper function to delete stored tokens

## Running the Example

```bash
cd examples/with_keyring
go run main.go
```

### First Run

On the first run, since no token is stored, the example will:
1. Start the GitHub OAuth device flow authentication
2. Display a URL and user code for authorization
3. Wait for you to complete the authorization in your browser
4. Store the token securely in your system keyring
5. Use the token to make a request to GitHub Copilot

### Subsequent Runs

On subsequent runs:
1. The example retrieves the stored token from the keyring
2. Uses the token directly without requiring re-authentication
3. Makes requests to GitHub Copilot

## How It Works

The example uses three main functions:

### `getTokenFromKeyring()`
Retrieves the stored GitHub token from the system keyring.

```go
token, err := getTokenFromKeyring()
if err != nil {
    // Token not found, need to authenticate
}
```

### `setTokenInKeyring(token)`
Stores a GitHub token securely in the system keyring.

```go
err := setTokenInKeyring(token)
if err != nil {
    // Handle error
}
```

### `DeleteToken()`
Removes the stored token from the keyring (useful for re-authentication or cleanup).

```go
err := DeleteToken()
if err != nil {
    // Handle error
}
```

## Platform Support

The `go-keyring` library supports:
- **macOS**: Uses Keychain
- **Windows**: Uses Credential Manager
- **Linux**: Uses Secret Service API (requires `gnome-keyring` or `kwallet`)

## Security Considerations

- Tokens are stored in the system's secure credential storage
- The token is associated with the service name `adk-copilot-llm` and user name `github-token`
- On Linux, ensure you have a keyring daemon running (e.g., `gnome-keyring-daemon`)

## Cleanup

To remove the stored token and re-authenticate:

```bash
# You can modify the example to call DeleteToken() before running, or
# Use system tools:

# macOS
security delete-generic-password -s "adk-copilot-llm" -a "github-token"

# Windows
cmdkey /delete:adk-copilot-llm:github-token

# Linux (with Secret Service)
secret-tool clear service adk-copilot-llm username github-token
```
