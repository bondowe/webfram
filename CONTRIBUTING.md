# Contributing to WebFram

Thank you for your interest in contributing to WebFram! This document provides guidelines and information about our CI/CD pipeline.

## Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests locally (`go test ./...`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## CI/CD Pipeline

Our CI pipeline runs automatically on every push and pull request. It includes:

### 1. Testing

- Runs on Go 1.22 and 1.23
- Executes all unit tests with race detection
- Generates code coverage reports
- Enforces a minimum coverage threshold of 70%

### 2. Building

- Verifies that all packages build successfully
- Tests the example applications

### 3. Linting

- Runs golangci-lint with comprehensive checks
- Ensures code quality and consistency

### 4. Coverage Reporting

- Uploads coverage to Codecov
- Generates detailed coverage reports in workflow summary
- Provides coverage badges on README

## Running Tests Locally

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# View coverage report
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out | grep total
```

## Running Linter Locally

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Run with specific timeout
golangci-lint run --timeout=5m
```

## Coverage Requirements

- **Minimum coverage**: 70%
- Coverage reports are generated for every PR
- Coverage must not decrease significantly with new changes

## Automated Issue Creation

When the CI pipeline fails on the main branch:

- An issue is automatically created with failure details
- If an issue already exists, a comment is added
- Issues are labeled with `ci-failure`, `bug`, and `automated`

## Pull Request Checks

All pull requests must pass:

- ✅ All tests on Go 1.22 and 1.23
- ✅ Build verification
- ✅ Linting checks
- ✅ Coverage threshold (70%)

## Code Style

We follow standard Go conventions:

- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Write clear, self-documenting code
- Add comments for exported functions and types
- Keep functions focused and concise

## Testing Guidelines

- Write tests for all new features
- Include both positive and negative test cases
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for high test coverage (70%+)

## Commit Message Guidelines

- Use clear, descriptive commit messages
- Start with a verb in present tense (Add, Fix, Update, etc.)
- Keep the first line under 72 characters
- Add detailed description if needed

Example:

```
Add JSON Patch validation support

- Implement validation after patch operations
- Add error handling for invalid patches
- Update tests to cover new validation logic
```

## Reporting Issues

When reporting issues, please include:

- Go version
- Operating system
- Steps to reproduce
- Expected behavior
- Actual behavior
- Any relevant logs or error messages

## Questions?

Feel free to open an issue for any questions or concerns about contributing.

Thank you for contributing to WebFram! 🚀
