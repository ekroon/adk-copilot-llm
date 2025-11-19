# Examples

This directory contains example applications demonstrating how to use the adk-copilot-llm library.

## auth_demo.go

Demonstrates the authentication flow with structured logging using `slog`.

### Usage

```bash
# Run with INFO level logging (default)
go run auth_demo.go

# Run with DEBUG level logging to see detailed information
LOG_LEVEL=DEBUG go run auth_demo.go

# Run with WARN level to see only warnings and errors
LOG_LEVEL=WARN go run auth_demo.go
```

### What you'll see

The demo shows:
- **INFO logs**: High-level flow events (starting auth, polling, success)
- **DEBUG logs**: Detailed information about each API call and polling attempt
- **WARN logs**: Rate limiting (slow_down) events with backoff information
- **ERROR logs**: Any failures that occur

Example output with DEBUG logging:
```
=== GitHub Copilot Authentication Demo ===
Log level: DEBUG

time=2025-11-19T13:00:00.000Z level=DEBUG msg="Starting device flow authentication" url=https://github.com/login/device/code
time=2025-11-19T13:00:00.100Z level=INFO msg="Device flow started successfully" verification_uri=https://github.com/login/device user_code=ABCD-1234 expires_in=900 interval=5

To authenticate with GitHub Copilot:
1. Visit: https://github.com/login/device
2. Enter code: ABCD-1234

Waiting for authorization...
time=2025-11-19T13:00:00.200Z level=INFO msg="Starting to poll for access token" initial_interval_seconds=5
time=2025-11-19T13:00:05.200Z level=DEBUG msg="Checking access token status"
time=2025-11-19T13:00:05.300Z level=DEBUG msg="Authorization still pending, continuing to poll"
...
time=2025-11-19T13:00:15.200Z level=INFO msg="Successfully obtained access token"

Successfully authenticated! Token length: 40
```

## main.go

Full example showing how to use the Copilot LLM for content generation.

### Usage

```bash
# With existing token
GITHUB_TOKEN=your_token_here go run main.go

# Without token (will initiate device flow)
go run main.go
```

This example demonstrates:
- Authentication (if no token provided)
- Non-streaming content generation
- Streaming content generation
- Multi-turn conversations

## Tips

- Use `LOG_LEVEL=DEBUG` when troubleshooting authentication issues
- If you see "slow_down" warnings, the library will automatically back off and retry
- The authentication token expires after a period, so you may need to re-authenticate
- Store your token securely and set it as an environment variable for convenience
