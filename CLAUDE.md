# Rocha

Tmux-based session manager for Claude Code CLI written in Go. Enables developers to manage multiple Claude conversation sessions, each with its own isolated git worktree and shell environment.

## Quick Component Location Guide

- **cmd/** - Entry point (main.go with version variables)
- **internal/cmd/** - Kong CLI commands (run, attach, status, setup, notify, etc.)
- **internal/domain/** - Domain entities and session state constants
- **internal/ports/** - Interface definitions (TmuxClient, SessionRepository, GitRepository, EditorOpener, SoundPlayer)
- **internal/services/** - Application services (session, git, shell, settings, notification, migration)
- **internal/ui/** - Bubble Tea TUI components (SessionList, SessionForm, Model, KeyMaps)
- **internal/config/** - Settings, paths, and Claude directory management
- **internal/logging/** - Structured logging (slog)
- **internal/adapters/tmux/** - Tmux abstraction layer (Client interface)
- **internal/adapters/storage/** - SQLite-based persistence (GORM)
- **internal/adapters/git/** - Git worktree operations
- **internal/adapters/editor/** - Platform-specific editor integration
- **internal/adapters/sound/** - Notification sounds
- **internal/adapters/process/** - Process inspection (reading command-line arguments from running processes)
- **test/integration/** - CLI integration tests

## Detailed Documentation

- **ARCHITECTURE.md** - System diagrams, data flows, component interactions, hook event mapping - load this file first everytime you are planing or need to learn about this topics
- **.claude/rules/** - All code conventions, commit guidelines, testing policies (automatically loaded)

## Development Workflow

### Building for Testing

Build with version info injection based on branch name:

```bash
go build -ldflags="-X main.Version=$(git branch --show-current)-v1 \
  -X main.Commit=$(git rev-parse HEAD) \
  -X main.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X main.GoVersion=$(go version | awk '{print $3}')" \
  -o ./bin/rocha-$(git branch --show-current)-v1 ./cmd
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
