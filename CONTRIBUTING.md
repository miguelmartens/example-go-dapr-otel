# Contributing

Thank you for your interest in contributing to this project.

## Development Setup

1. **Clone the repository**

   ```bash
   git clone https://github.com/miguelmartens/example-go-dapr-otel.git
   cd example-go-dapr-otel
   ```

2. **Ensure you have the requirements**

   - Go 1.26+
   - [golangci-lint](https://golangci-lint.run/) (for `make lint`)
   - [Prettier](https://prettier.io/) (for `make prettier`)

3. **Run the app locally**

   ```bash
   make dev
   ```

## Development Workflow

- `make build` – Build the binary
- `make test` – Run tests
- `make lint` – Run `go vet` and golangci-lint
- `make fmt` – Format Go code
- `make prettier` – Format JSON, YAML, Markdown
- `make tidy` – Tidy `go.mod`

## Pull Requests

1. Fork the repository and create a branch from `main`
2. Make your changes
3. Ensure tests pass: `make test`
4. Ensure lint passes: `make lint`
5. Format code: `make fmt` and `make prettier`
6. Open a pull request with a clear description of the changes
7. Link any related issues

## Code Style

- Follow standard Go conventions and [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` / `go fmt` for formatting
- Keep commits focused and well-described

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
