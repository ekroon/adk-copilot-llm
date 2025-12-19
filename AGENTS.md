# Agent Guidelines for adk-copilot-llm

## Build/Test Commands
- Run all tests: `go test -v ./...` or `make test`
- Run single test: `go test -v ./copilot -run TestName`
- Run vet (linting): `go vet ./...` or `make vet`
- Install deps: `go mod download` or `make setup`
- Run example: `cd examples && GITHUB_TOKEN=your_token go run main.go`

## Code Style
- **Language**: Go 1.24+
- **Imports**: Standard library first, external packages second, blank line between groups
- **Formatting**: Use `gofmt` (tabs for indentation)
- **Types**: Exported types have package doc comments. Struct fields documented inline when needed
- **Naming**: Use camelCase for unexported, PascalCase for exported. Acronyms stay uppercase (e.g., `HTTPClient`, `apiKeyURL`)
- **Error handling**: Always wrap errors with context using `fmt.Errorf("description: %w", err)`
- **Constants**: Group related constants together, use `const` blocks with comments
- **Interfaces**: Follow standard Go interface patterns (e.g., `model.LLM` from adk-go)
- **Concurrency**: Use `sync.RWMutex` for thread-safe shared state, prefer read locks when possible
- **Testing**: Use table-driven tests with `t.Run()` for subtests. Test both success and error cases
