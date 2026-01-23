# Contributing to MindBalancer

First off, thank you for considering contributing to MindBalancer! It's people like you that make MindBalancer such a great tool.

## Code of Conduct

By participating in this project, you are expected to uphold our Code of Conduct: be respectful, inclusive, and considerate of others.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples** (config files, SQL commands, API requests)
- **Describe the behavior you observed and what you expected**
- **Include logs** with `log_level = debug` if possible
- **Include your environment** (OS, Go version, MindBalancer version)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear and descriptive title**
- **Provide a detailed description of the proposed functionality**
- **Explain why this enhancement would be useful**
- **List any alternatives you've considered**

### Pull Requests

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code follows the existing style
6. Issue that pull request!

## Development Setup

### Prerequisites

- Go 1.21 or later
- SQLite3
- Make

### Getting Started

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/mindbalancer.git
cd mindbalancer

# Add upstream remote
git remote add upstream https://github.com/mindbalancer/mindbalancer.git

# Install dependencies and dev tools
make dev-setup

# Run tests
make test

# Build
make build

# Run locally
make run
```

### Project Structure

```
mindbalancer/
├── cmd/
│   ├── mindbalancer/      # Main server binary
│   └── mindsql/           # CLI client binary
├── internal/
│   ├── admin/             # Admin interface
│   ├── balancer/          # Load balancing logic
│   ├── circuit/           # Circuit breaker
│   ├── config/            # Configuration management
│   ├── health/            # Health checking
│   ├── metrics/           # Prometheus metrics
│   ├── provider/          # AI provider adapters
│   ├── proxy/             # API proxy
│   ├── router/            # Request routing
│   └── storage/           # SQLite storage
├── pkg/
│   └── protocol/          # MySQL protocol implementation
├── api/
│   └── openai/            # OpenAI API types
├── configs/               # Example configurations
├── grafana/               # Grafana dashboards
└── scripts/               # Build and utility scripts
```

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write descriptive commit messages
- Add comments for exported functions

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test -v ./internal/balancer/...

# Run with race detector
go test -race ./...
```

### Commit Messages

We follow conventional commits:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting
- `refactor`: Code restructuring
- `test`: Adding tests
- `chore`: Maintenance

Examples:
```
feat(balancer): add latency-based load balancing algorithm
fix(provider): handle Anthropic rate limit errors correctly
docs(readme): update quick start guide
```

### Adding a New Provider

1. Create a new file in `internal/provider/`
2. Implement the `Provider` interface
3. Add the provider to the `New()` factory function
4. Add tests in `internal/provider/provider_test.go`
5. Update documentation

Example:

```go
// internal/provider/newprovider.go
package provider

type NewProvider struct {
    *BaseProvider
}

func NewNewProvider(server storage.Server, timeout time.Duration) *NewProvider {
    return &NewProvider{
        BaseProvider: NewBaseProvider(server, timeout),
    }
}

func (p *NewProvider) Name() string {
    return "newprovider"
}

// Implement remaining Provider interface methods...
```

### Adding a New mindsql Command

1. Add the SQL pattern handling in `internal/admin/admin.go`
2. Implement the execution function
3. Add to help text in `cmd/mindsql/main.go`
4. Add tests

## Review Process

1. All submissions require review
2. We use GitHub pull requests for this purpose
3. A maintainer will review your PR within a few days
4. Address any feedback and push changes
5. Once approved, a maintainer will merge your PR

## Release Process

Releases are managed by maintainers:

1. Version bump in code
2. Update CHANGELOG.md
3. Create git tag
4. GitHub Actions builds and publishes releases

## Getting Help

- 💬 [Discord](https://discord.gg/mindbalancer)
- 📧 [Email](mailto:dev@mindbalancer.io)
- 🐛 [Issues](https://github.com/mindbalancer/mindbalancer/issues)

## Recognition

Contributors are recognized in:
- GitHub contributors page
- Release notes
- Special thanks section in README (for significant contributions)

Thank you for contributing! 🎉
