# Contributing to gai

Thank you for your interest in contributing to gai! We welcome contributions from the community and are grateful for any help you can provide.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/gai.git
   cd gai
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/recera/gai.git
   ```
4. Create a new branch for your feature or fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

1. Ensure you have Go 1.21 or later installed
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Run tests:
   ```bash
   make test
   ```
4. Run linter:
   ```bash
   make lint
   ```

## Making Changes

1. **Write tests**: All new features and bug fixes should have tests
2. **Follow Go conventions**: Use `gofmt` and follow standard Go naming conventions
3. **Keep commits focused**: Each commit should represent a single logical change
4. **Update documentation**: Keep README and comments up to date with your changes

## Code Style

- Use `gofmt` to format your code
- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Add comments to exported functions, types, and packages
- Keep line length under 120 characters when possible

## Testing

- Write unit tests for new functionality
- Ensure all tests pass: `make test`
- Add integration tests when appropriate (use build tags)
- Mock external dependencies in tests

## Submitting Changes

1. Push your changes to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```
2. Create a Pull Request on GitHub
3. Describe your changes in detail
4. Link any related issues

## Pull Request Guidelines

- **Title**: Use a clear, descriptive title
- **Description**: Explain what changes you made and why
- **Testing**: Describe how you tested your changes
- **Breaking changes**: Clearly mark any breaking changes

## Adding a New Provider

If you're adding support for a new LLM provider:

1. Create a new file in `providers/` (e.g., `providers/newprovider.go`)
2. Implement the `ProviderClient` interface
3. Add the provider to the client initialization in `llm_client.go`
4. Add tests for the new provider
5. Update the README with the new provider information
6. Add example usage in `_examples/`

## Reporting Issues

- Use the GitHub issue tracker
- Check if the issue already exists
- Provide a clear description and steps to reproduce
- Include Go version and OS information
- Add code examples if applicable

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Assume good intentions

## Questions?

Feel free to open an issue for any questions about contributing!