## Development and Testing

### Code Structure

The project follows a standard Go project layout:
- `cmd/go-acme-dns-manager/`: Main application entry point
- `internal/manager/`: Core business logic modules
- `.github/workflows/`: CI/CD pipeline configurations

### Running Tests

The project includes comprehensive unit tests and integration tests. For details on the testing approach, refer to [TESTING.md](TESTING.md).

Run basic unit tests with:

```bash
make test
```

Run all tests including integration tests:

```bash
make test-all
```

Or manually:

```bash
# Unit tests only
go test ./...

# All tests including integration tests
RUN_INTEGRATION_TESTS=1 go test ./...
```

### Code Quality Tools

The project uses several code quality tools to maintain high standards:

1. **Linting**: Run the linter to check code style and potential issues:
   ```bash
   make lint
   ```

2. **Building**: Build the application:
   ```bash
   make build
   ```

3. **All Checks**: Run all quality checks in sequence:
   ```bash
   make all
   ```

### Constants and Configurations

Common values are stored as constants in `internal/manager/constants.go` rather than being hard-coded throughout the application. These include:

- File permissions
- Default timeout values
- Grace periods for certificate renewal

### Testing Contributions

When contributing, please ensure:

1. All existing tests pass
2. New features include appropriate tests
3. Code follows Go best practices
4. Changes are documented in CHANGES.md
