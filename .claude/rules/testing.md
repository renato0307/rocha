# Testing Guidelines

## When to write Unit Tests

ALWAYS but we seek for writting tests
for things that are worthwhile not just for
coverage per se.

Always write tests for a bug, before you fix it.

## When Planning

Include what tests you will include and which you don't and why.

## Running Unit Tests

```bash
# Run all unit tests
go test ./internal/...

# Run tests with verbose output
go test ./internal/... -v

# Run tests with coverage
go test ./internal/... -cover

# Run specific test
go test ./internal/services/... -run TestCreateSession
```

## Test Patterns

### Table-Driven Tests (Preferred)

Use table-driven tests for functions with multiple scenarios:

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        expected  int
        assertErr assert.ErrorAssertionFunc
    }{
        {
            name:      "valid input",
            input:     "hello",
            expected:  5,
            assertErr: assert.NoError,
        },
        {
            name:      "empty input returns error",
            input:     "",
            expected:  0,
            assertErr: assert.Error,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionName(tt.input)

            tt.assertErr(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Single Scenario Tests

For tests with mocks or complex setup:

```go
func TestFunctionName_Scenario(t *testing.T) {
    // Create mocks
    gitRepo := portsmocks.NewMockGitRepository(t)

    // Setup expectations
    gitRepo.EXPECT().Method(mock.Anything).Return(value, nil)

    // Create service
    service := NewService(gitRepo)

    // Execute
    result, err := service.Method(...)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

## Mocks

Mocks are generated using mockery. Config is in `.mockery.yaml`.

```bash
# Regenerate mocks after adding interfaces
mockery
```

## Test Coverage

Focus unit tests on:
- Pure functions (validation, sanitization, parsing)
- Service layer with mocked dependencies
- Error handling paths

Integration tests run in Docker (see testing_safety.md).
