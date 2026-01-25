# Integration Test Safety

## NEVER use `test-integration-local-dangerous`

Coding agents (Claude, Copilot, etc.) must NEVER run:
- `make test-integration-local-dangerous`
- `cd test/integration && go test ...`

These commands run tests directly on the host system and can:
- Modify shell configuration files
- Create/delete git worktrees
- Modify tmux sessions
- Write to arbitrary paths

## Always use Docker

Use `make test-integration` which runs tests safely in a container.
