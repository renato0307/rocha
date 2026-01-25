# Integration Test Safety

## CRITICAL: NEVER run integration tests directly

Coding agents (Claude, Copilot, etc.) must NEVER run:
- `go test ./test/integration/...` ← FORBIDDEN
- `go test ./test/integration/... -run TestName` ← FORBIDDEN
- `cd test/integration && go test ...` ← FORBIDDEN
- `make test-integration-local-dangerous` ← FORBIDDEN

These commands run tests directly on the host system and can:
- Modify shell configuration files (~/.zshrc, ~/.bashrc)
- Create/delete git worktrees
- Modify tmux sessions
- Write to arbitrary paths
- Corrupt the user's rocha installation

## Always use make with Docker

```bash
make test-integration                           # Run all integration tests (Docker)
make test-integration-verbose                   # Run with verbose output (Docker)
make test-integration-run TEST=TestSettingsKeys # Run specific test (Docker)
```

These commands run tests safely in an isolated Docker container.
