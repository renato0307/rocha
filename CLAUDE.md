# Rocha

Tmux-based session manager for Claude Code CLI written in Go. Enables developers to manage multiple Claude conversation sessions, each with its own isolated git worktree and shell environment.

## Quick Component Location Guide

- **cmd/** - CLI commands (run, attach, status, setup, notify, etc.)
- **ui/** - Bubble Tea TUI components (SessionList, SessionForm, Model, KeyMaps)
- **tmux/** - Tmux abstraction layer (Client interface, DefaultClient)
- **storage/** - SQLite-based persistence (GORM, sessions/flags/statuses tables)
- **git/** - Git worktree operations (create, remove, branch validation)
- **editor/** - Platform-specific editor integration
- **state/** - Session state enum (Idle, Working, Waiting, Exited)
- **operations/** - High-level business logic operations
- **sound/** - Notification sounds
- **logging/** - Structured logging (slog)
- **version/** - Build metadata (version, commit, date)
- **paths/** - Centralized path computation (uses ROCHA_HOME)
- **config/** - Settings and Claude directory management

## Detailed Documentation

- **ARCHITECTURE.md** - System diagrams, data flows, component interactions, hook event mapping - load this file first everytime you are planing or need to learn about this topics
- **.claude/rules/** - All code conventions, commit guidelines, testing policies (automatically loaded)

## Development Workflow

### Building for Testing

Build with version info injection based on branch name:

```bash
go build -ldflags="-X rocha/version.Version=$(git branch --show-current)-v1 \
  -X rocha/version.Commit=$(git rev-parse HEAD) \
  -X rocha/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X rocha/version.GoVersion=$(go version | awk '{print $3}')" \
  -o ~/.local/bin/rocha-$(git branch --show-current)-v1
```

### Running with Debug

```bash
rocha --debug --debug-file <filename>
rocha run --dev  # Use --dev flag with run command only
```

### Testing Builds

Always use `run --dev` flag when testing built binaries to verify version info in dialog headers.

### Post-Development

After adding new packages/components, update ARCHITECTURE.md diagrams (minimize text, maximize visuals).

## Environment Variables

- **ROCHA_HOME** - Base directory for rocha data (default: `~/.rocha`)
- **ROCHA_EDITOR** - Preferred editor (overrides `$VISUAL` and `$EDITOR`)

## External Dependencies

- **tmux** (required) - Terminal multiplexer
- **git** (required) - Version control for worktrees
- **claude** (auto-bootstrapped) - Claude Code CLI
- **editor** (optional) - VS Code or any text editor
