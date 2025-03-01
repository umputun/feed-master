# Feed Master Development Guidelines

## Build, Test, Lint Commands
```
# Run tests
cd app && go test -race -v ./...                            # Run all tests
cd app && go test -race -v ./path/to/package                # Test specific package
cd app && go test -race -v ./path/to/package -run TestName  # Run specific test

# Lint code
golangci-lint run ./...                                     # Lint entire codebase
golangci-lint run ./path/to/package                         # Lint specific package

# Install golangci-lint if needed
# brew install golangci-lint                                # macOS
# go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.58.0  # Any OS with Go

# Build application
cd app && go build -o feed-master                           # Build binary
```

## Code Style Guidelines
- **Imports**: Standard Go grouping (stdlib, external, project)
- **Error Handling**: Use descriptive errors with pkg/errors for wrapping
- **Naming**: Follow Go conventions (CamelCase for exported, camelCase for private)
- **Types**: Use strong typing, prefer interfaces for flexibility
- **Code Structure**: Group related functionality in packages
- **Tests**: Write unit tests for all exported functions
- **Comments**: Document exported functions following Go conventions
- **Error Reporting**: Use lgr for logging with appropriate levels
- **Code Format**: Always run `gofmt -s` before committing
- **Linting**: Follow rules in .golangci.yml configuration
  - Project uses 15+ linters including govet, staticcheck, gosec
  - Run `golangci-lint run --verbose` to see detailed issues

## Project-Specific Patterns
- Use go-pkgz libraries where appropriate for common functionality
- Follow error wrapping conventions with pkg/errors
- Prefer dependency injection for testability
- Handle context properly for cancellation