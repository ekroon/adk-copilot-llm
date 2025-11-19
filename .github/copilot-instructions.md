# GitHub Copilot Instructions for adk-copilot-llm

## Logging Guidelines

### Use `log/slog` for Non-User-Essential Output

This project uses Go's structured logging package `log/slog` for all non-user-essential logging. User-facing output should continue to use `fmt.Printf` or similar.

**When to use slog:**
- Debug information for developers
- Internal state tracking
- API request/response logging
- Error details for troubleshooting
- Performance metrics
- Background operations

**When to use fmt.Printf:**
- User instructions (e.g., "Visit: https://...")
- User prompts
- Progress messages that users need to see
- Final results/output

### Log Levels

Use appropriate log levels:

- **`slog.Debug`**: Detailed information for debugging, typically only relevant during development
  - Example: API request details, polling attempts, internal state changes
  
- **`slog.Info`**: General informational messages about normal operations
  - Example: Starting authentication, successful completion, configuration details
  
- **`slog.Warn`**: Warning messages about potentially problematic situations that don't prevent operation
  - Example: Slow down errors, retries, fallback behaviors
  
- **`slog.Error`**: Error messages about failures that should be investigated
  - Example: API failures, invalid responses, authentication errors

### Structured Logging

Always use structured logging with key-value pairs for better searchability and debugging:

```go
// Good - structured logging
slog.Info("Starting authentication", "url", authURL, "timeout", timeout)
slog.Warn("Received slow_down error", "new_interval_seconds", interval.Seconds())
slog.Error("Failed to fetch API key", "status", statusCode, "error", err)

// Bad - string formatting
slog.Info(fmt.Sprintf("Starting authentication at %s with timeout %d", authURL, timeout))
```

### Error Logging

When logging errors:
1. Always include the error object in the log
2. Add relevant context with structured fields
3. Use appropriate log level (Error for failures, Warn for recoverable issues)

```go
// Good
if err != nil {
    slog.Error("Failed to decode response", "error", err, "status_code", resp.StatusCode)
    return fmt.Errorf("failed to decode response: %w", err)
}

// Also good - for warnings
if strings.Contains(err.Error(), "slow_down") {
    slog.Warn("Received slow_down error, backing off", 
        "error", err, 
        "new_interval_seconds", newInterval.Seconds())
    // ... handle backoff ...
}
```

## Testing

- Write tests for new functionality
- Use table-driven tests for multiple scenarios
- Mock external dependencies (HTTP clients, etc.)
- Test error conditions as well as happy paths

## Code Style

- Follow standard Go conventions
- Use meaningful variable names
- Add comments for exported functions and types
- Keep functions focused and small
- Handle errors explicitly, don't ignore them
