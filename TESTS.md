# Testing Guide

This document describes how to run tests for the Tester-API project and provides guidance for local development and CI integration.

## Overview

The Tester-API has been refactored to support comprehensive unit testing without requiring Gin or GORM at test runtime. The implementation uses:

- **Store abstraction**: A `Store` interface that decouples business logic from database implementation
- **MemoryStore**: An in-memory, thread-safe implementation for fast, deterministic tests
- **GormStore**: Production adapter that wraps GORM database operations
- **Business logic layer**: Framework-agnostic functions in `internal/api` that are the primary test targets

## Prerequisites

- Go toolchain (Go 1.20 or later)
- Network access for downloading dependencies (if modules are not vendored)

**Note**: The environment used for planning this feature lacks network access and Go toolchain. Tests must be run in a developer environment or CI system with Go installed.

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Tests with Verbose Output

```bash
go test ./... -v
```

### Run Tests with Coverage Report

```bash
go test ./... -cover
```

### Run Specific Package Tests

```bash
# Test only the API business logic
go test ./internal/api -v

# Test only the store implementations
go test ./internal/store -v
```

### Run Specific Test Functions

```bash
# Run only validation tests
go test ./internal/api -run TestCreatePing_ValidationError -v

# Run only success path tests
go test ./internal/api -run Success -v
```

## Test Architecture

### Unit Tests (internal/api)

The core business logic tests use only:
- Standard library (`testing`, `sync`)
- Internal packages (`internal/store`, `internal/api`)
- **No Gin or GORM imports** - tests are fast and deterministic

Test coverage includes:

1. **TestCreatePing_Success**: Validates successful ping creation with valid input
2. **TestCreatePing_ValidationError**: Tests validation failures for empty/whitespace messages
3. **TestCreatePing_StorageError**: Tests error propagation from storage layer
4. **TestListPings_ReturnsSeeded**: Validates retrieval of existing records
5. **TestListPings_EmptyStore**: Tests empty result set behavior
6. **TestCreatePing_TrimsWhitespace**: Validates whitespace trimming

### Storage Error Injection

The `MemoryStore` provides an error injection mechanism for testing error handling:

```go
// Example: Test storage failure handling
memStore := store.NewMemoryStore()
memStore.SetErrorOnSave(errors.New("simulated storage failure"))

err := api.CreatePing(memStore, &store.Ping{Message: "test"})
// err will be the simulated error
```

### Seeding Test Data

For tests that require pre-existing data:

```go
memStore := store.NewMemoryStore()
seededPings := []store.Ping{
    {Message: "first"},
    {Message: "second"},
}
memStore.Seed(seededPings)

pings, _ := api.ListPings(memStore)
// pings will contain the seeded data
```

## Local Development Features

### USE_IN_MEMORY_STORE Environment Variable

For local experimentation only, you can configure the application to use the in-memory store instead of SQLite:

```bash
export USE_IN_MEMORY_STORE=true
go run main.go
```

**⚠️ Important**: This is for local development and testing only. Do NOT enable this in production. The default production configuration uses `GormStore` backed by SQLite.

## CI Integration

While this plan does not modify CI workflows, here's how to integrate these tests into your CI pipeline:

### GitHub Actions Example

```yaml
- name: Run tests
  run: go test ./... -v

- name: Run tests with coverage
  run: go test ./... -coverprofile=coverage.out

- name: Display coverage
  run: go tool cover -func=coverage.out
```

### Jenkins Example

```groovy
stage('Test') {
    steps {
        sh 'go test ./... -v'
    }
}
```

## Code Structure

```
.
├── internal/
│   ├── store/
│   │   ├── ping.go          # Shared Ping model
│   │   ├── store.go         # Store interface
│   │   ├── mem_store.go     # In-memory implementation (tests)
│   │   └── gorm_store.go    # GORM implementation (production)
│   └── api/
│       ├── api.go           # Business logic functions
│       └── api_test.go      # Unit tests
└── main.go                  # HTTP handlers and wiring
```

## Validation Rules

The business logic enforces the following validation rules:

1. **Message Required**: Ping messages must be non-empty after trimming whitespace
   - Empty strings return `ErrEmptyMessage` (400 Bad Request)
   - Whitespace-only strings are trimmed and treated as empty

2. **Storage Errors**: Any storage operation failure returns an error
   - Mapped to 500 Internal Server Error in HTTP handlers

## Troubleshooting

### "cannot find package" errors

Run `go mod download` to fetch dependencies:

```bash
go mod download
go test ./...
```

### Tests fail in environment without network

If your environment lacks network access:
1. Pre-download dependencies in an environment with network access
2. Vendor dependencies: `go mod vendor`
3. Run tests with vendored dependencies: `go test ./... -mod=vendor`

### Database locked errors

If you get "database is locked" errors:
- Ensure no other process is using the SQLite database
- Use in-memory store for tests: `go test ./internal/api` (API tests don't touch the database)
- The unit tests in `internal/api` do not require database access

## Best Practices

1. **Run tests frequently**: Execute `go test ./internal/api -v` after making changes
2. **Test in isolation**: Use `MemoryStore` to avoid database side effects
3. **Use table-driven tests**: Add new validation test cases to existing table-driven tests
4. **Mock storage failures**: Use `SetErrorOnSave()` to test error handling paths
5. **Keep tests fast**: Avoid sleep, network calls, or database access in unit tests

## Additional Resources

- Go testing documentation: https://golang.org/pkg/testing/
- Go modules documentation: https://golang.org/ref/mod
- GORM documentation: https://gorm.io/docs/
- Gin documentation: https://gin-gonic.com/docs/
