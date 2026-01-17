# Agent Guidelines for adk-copilot-llm

## Purpose
This repository provides a Go implementation of the ADK `model.LLM` interface
that uses the GitHub Copilot CLI via the official copilot SDK.
Agents should keep changes minimal and aligned with existing patterns.

## Repo Layout
- `copilot/`: core LLM implementation and unit tests.
- `examples/`: runnable sample apps (basic + tools).
- `README.md` / `GETTING_STARTED.md`: public docs and usage notes.

## Build, Lint, Test
- Install dependencies: `go mod download` or `make setup`.
- Build library package: `go build ./copilot`.
- Run all tests: `go test -v ./...`.
- Run copilot package tests: `go test -v ./copilot/...`.
- Run a single test: `go test -v ./copilot -run TestName`.
- Run vet (lint): `go vet ./...` or `make vet`.
- Run basic example: `cd examples && GITHUB_TOKEN=... go run main.go`.
- Run tools example: `cd examples/tools && GITHUB_TOKEN=... go run main.go`.
- Build tools example: `cd examples/tools && go build .`.

## Configuration Defaults
- Default model: `gpt-4`.
- Default log level: `error`.
- Default CLI path: `copilot` or `COPILOT_CLI_PATH`.
- Streaming defaults to `false` unless configured.

## Environment Variables
- `COPILOT_CLI_PATH`: overrides the Copilot CLI binary path.
- `GITHUB_TOKEN`: required for examples; must include Copilot access.
- `COPILOT_CLI_PATH` from env is ignored when `Config.CLIPath` is set.

## Code Style
### Formatting
- Use `gofmt` for all Go files (tabs for indentation).
- Keep spacing and alignment consistent with existing files.
- Avoid manual alignment that `gofmt` would remove.

### Imports
- Standard library imports first.
- Third-party imports second.
- Keep exactly one blank line between import groups.
- Remove unused imports instead of using blank identifiers.

### Naming
- Exported identifiers use PascalCase.
- Unexported identifiers use camelCase.
- Initialisms stay uppercase (e.g., `HTTPClient`, `LLMRequest`).
- Keep receiver names short but clear (`c` for CopilotLLM, `tc` for toolContext).
- Avoid overly terse variable names outside established receiver patterns.

### Types and Structs
- Exported types require doc comments.
- Document struct fields when behavior is non-obvious.
- Prefer typed structs over `map[string]any`.
- Use struct tags consistently for JSON/schema definitions.

### Error Handling
- Wrap errors with context: `fmt.Errorf("description: %w", err)`.
- Return early on failure; avoid nested `if` chains.
- Include helpful context (IDs, model names, paths).
- Avoid swallowing errors unless explicitly intentional.

### Context and Resources
- Accept `context.Context` in public entrypoints.
- Pass contexts through to SDK calls.
- Call `defer llm.Close()` immediately after `copilot.New()`.
- Keep `Close` idempotent and safe to call once.

### Concurrency
- Protect shared state with `sync.Mutex` or `sync.RWMutex`.
- Prefer read locks for read-heavy paths.
- Keep lock scope minimal and avoid long-running work inside locks.

## Testing Guidance
- Use table-driven tests with `t.Run()`.
- Cover success and error paths.
- Use `t.Helper()` in shared helpers.
- Keep tests deterministic and free of network dependencies.
- Tests live next to implementation in the `copilot` package.

## Copilot LLM Implementation Notes
- `CopilotLLM` lazily starts the CLI client via `ensureStarted()`.
- `GenerateContent` selects `req.Model` when provided.
- The `stream` argument overrides `config.Streaming` when true.
- Sessions are created per request and closed after use.

## LLM Response Handling
- The iterator yields `(response, error)`; always check `err`.
- `resp.Content` can be nil; guard before iterating parts.
- For streaming, output tokens as they arrive.
- Stop processing when `resp.TurnComplete` is true.
- Avoid returning partial responses on error.

## Tool Integration Notes
- Tools must implement `tool.Tool` plus `Declaration()` and `Run()`.
- Prefer `functiontool.New()` for type-safe schema definitions.
- Validate tool inputs/outputs using JSON schema tags.
- The standalone `toolContext` has limited runtime features:
  - `Agent()`, `Session()`, `Actions()` return nil.
  - `SearchMemory()` returns an error.
- For full runtime features, use `llmagent.New()` in adk-go.

## Tool Conversion Flow
- Validate tool implements required interfaces.
- Convert `genai.FunctionDeclaration` to Copilot parameters.
- Wrap handler to build `toolContext` with call ID.
- Run tool via `tool.Run(ctx, args)`.
- Marshal output to JSON for Copilot SDK.
- Return errors to the LLM if handler fails.

## Tool Schema Guidelines
- Use `jsonschema` tags for enums and validation.
- Keep input/output structs small and typed.
- Return a `map[string]any` only when required.

## Prompt Formatting
- `formatPrompt` maps `model` role to `Assistant`.
- `system` content is prefixed with `System:`.
- Multi-turn conversation inserts blank lines between turns.
- Keep prompt formatting stable when modifying prompt logic.

## Logging and Debugging
- Use `config.LogLevel` to tune CLI logs (`error`, `warn`, `info`, `debug`).
- Prefer structured context in error messages over extra logging.
- Avoid printing directly in library code.

## Go Module Conventions
- Go version is `1.24.10` (see `go.mod`).
- Add dependencies only when required.
- Run `go mod tidy` only if dependencies change.

## Documentation
- Update `README.md` or `GETTING_STARTED.md` when public APIs change.
- Keep example code in sync with API changes.
- Mention new environment variables or config fields in docs.

## Examples and CLI
- Examples rely on the GitHub Copilot CLI being installed.
- The Copilot CLI is managed by `gh copilot`.
- If docs or tests mention the CLI, note required authentication.

## Change Hygiene
- Keep edits focused on the requested behavior.
- Avoid refactors unless they directly support the change.
- Keep diff size minimal; follow existing patterns.

## Quick Reference
- Tests: `go test -v ./...`.
- Single test: `go test -v ./copilot -run TestName`.
- Vet: `go vet ./...`.
- Build: `go build ./copilot`.
