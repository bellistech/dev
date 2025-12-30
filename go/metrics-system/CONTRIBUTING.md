# Contributing to Metrics Collection System

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/metrics-system.git`
3. Create a feature branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Run tests: `make test`
6. Commit your changes: `git commit -am 'Add my feature'`
7. Push to your fork: `git push origin feature/my-feature`
8. Create a Pull Request

## Development Setup

### Prerequisites

- Go 1.21 or later
- PostgreSQL 12+ with TimescaleDB
- Protocol Buffers compiler (protoc)
- Make

### Quick Setup

```bash
./scripts/setup.sh
```

Or manually:

```bash
make setup
make proto
make build
```

## Code Style

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting: `make fmt`
- Run linter: `make lint`
- Maximum line length: 120 characters
- Use meaningful variable names

### Comments

- Add package comments for all packages
- Document exported functions, types, and constants
- Use complete sentences

## Adding a New Collector

1. Create `internal/agent/collector/yourname.go`
2. Implement the `Collector` interface
3. Register in `cmd/agent/main.go`
4. Add configuration option in `configs/agent.yaml`
5. Write tests: `internal/agent/collector/yourname_test.go`
6. Update documentation

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update the CHANGELOG.md with your changes
3. Ensure all tests pass
4. Request review from maintainers

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow

## Questions?

- Open a GitHub issue for bugs
- Start a discussion for feature requests
- Read the documentation in `docs/`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
