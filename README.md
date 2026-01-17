# Rocha - Claude Code Session Manager

A terminal UI application for managing multiple Claude Code sessions in tmux. Create named sessions, switch between them from a list view, and use tmux's native detach (Ctrl+B D) to return to the list.

## Features

- **List View**: See all active Claude Code sessions in a clean interface
- **Create Sessions**: Press `n` to create a new Claude Code session with a custom name
- **Attach to Sessions**: Press Enter to attach to a session (full Claude Code interaction)
- **Detach to List**: Use Ctrl+B then D (release Ctrl+B first, then press D) OR press Ctrl+Q to return to the list
- **Kill Sessions**: Press `x` to permanently kill a session
- **Sound Notifications**: Automatically plays sounds when Claude needs your attention:
  - **Stop**: Claude finishes and is waiting for your input (Glass/Complete sound)
  - **Notification**: Claude sends any notification (e.g., AskUserQuestion prompts) (Glass/Complete sound)
  - **Start**: Session initialized and ready (Submarine/Login sound)
  - **Prompt**: Silent (state tracked but no sound - you already know you submitted it)

## Installation

```bash
go build -o rocha
cp rocha ~/.local/bin/  # Or add to your PATH
```

## Usage

Rocha provides multiple commands for different use cases:

### Default TUI Mode
```bash
rocha  # Starts the interactive TUI
rocha --debug  # Start with debug logging enabled
```

### CLI Commands
```bash
rocha setup                    # Configure tmux status bar integration automatically
rocha status                   # Show session state counts (for tmux status bar)
rocha start-claude [args...]   # Start Claude with hooks (auto-called by TUI)
rocha play-sound               # Test notification sound
rocha notify <session> <event> # Called by Claude hooks (stop, prompt, start)
```

### Debug Logging

Enable debug logging to troubleshoot issues:
```bash
rocha --debug  # or -d
rocha --debug-file=/path/to/custom.log  # Custom log file path
rocha --debug --max-log-files=500  # Change max log files (default: 1000)
```

#### Default Behavior (--debug)
- Creates a log file with UUID name: `<uuid>.log`
- Logs are stored in OS-specific location:
  - **Linux**: `~/.local/state/rocha/`
  - **macOS**: `~/Library/Logs/rocha/`
  - **Windows**: `%LOCALAPPDATA%\rocha\logs\`
- Automatically rotates logs when limit is reached (default: 1000 files)
- Deletes oldest log files first when rotating

#### Custom Log File (--debug-file)
- Write logs to a specific file path
- Disables automatic log rotation and cleanup
- Useful for CI/CD or when you need persistent logs

#### Log Rotation
- `--max-log-files=N` - Keep at most N log files (default: 1000)
- `--max-log-files=0` - Unlimited logs (no rotation)
- Only applies when using `--debug` (not `--debug-file`)
- Rotation happens before creating each new log file

#### Log Format
Uses JSON format for structured logging:
```json
{"time":"2026-01-17T00:04:39.284089482Z","level":"INFO","msg":"Notification hook triggered","session":"s1","event":"stop"}
```

Includes detailed information about:
- Hook events (stop, prompt, start)
- State updates and session management
- Sound notifications
- Execution IDs and error conditions

### Help
```bash
rocha --help
rocha <command> --help
```

## Key Bindings

### List View
- `↑` or `k` - Move cursor up
- `↓` or `j` - Move cursor down
- `n` - Create new session
- `Enter` - Attach to selected session
- `x` - Kill selected session
- `q` or `Ctrl+C` - Quit application

### Attached to Session
- `Ctrl+B then D` - Detach and return to list (press Ctrl+B, release, then press D)
- `Ctrl+Q` - Quick detach shortcut (alternative to Ctrl+B D)
- All other tmux key bindings work normally

### Creating Session
- Type the session name
- `Enter` - Create the session
- `Esc` - Cancel

## Status Bar Integration

Rocha can display real-time session state counts in your tmux status bar, showing how many Claude sessions are waiting for input vs actively working.

### Quick Setup

Run the automatic setup command:
```bash
./rocha setup
```

This will:
- Add rocha's directory to your PATH in `~/.zshrc` or `~/.bashrc`
- Add the status bar configuration to your `~/.tmux.conf`
- Reload tmux configuration if tmux is running

The setup command is **idempotent** - you can run it multiple times safely without creating duplicates.

After running setup, reload your shell:
```bash
source ~/.zshrc  # or ~/.bashrc
```

### Manual Setup

Alternatively, you can manually:

1. Add rocha's directory to your PATH in `~/.zshrc` or `~/.bashrc`:
   ```bash
   export PATH="/path/to/rocha:$PATH"
   ```

2. Add this to your `~/.tmux.conf`:
   ```bash
   # Show Claude session states in status bar
   set -g status-right "Claude: #(rocha status) | %H:%M"
   set -g status-interval 1  # Update every 1 second
   ```

3. Reload your shell and tmux:
   ```bash
   source ~/.zshrc  # or ~/.bashrc
   tmux source-file ~/.tmux.conf
   ```

### Output Format

The status command shows:
- **W:N** - Number of sessions **waiting** for input (Claude finished)
- **A:N** - Number of sessions **actively working** (Claude processing)

Examples:
- `W:? A:?` - Rocha just started, no state reported yet
- `W:0 A:0` - No active sessions (or all sessions from previous run)
- `W:2 A:1` - 2 sessions waiting for input, 1 actively working
- `W:3 A:0` - 3 sessions waiting, none working

### How State Tracking Works

1. **Reset on TUI Start**: Every time you run `rocha`, a new execution ID is generated and state is reset
2. **Hook Updates**: When Claude events fire (stop/prompt/start), session states are updated
3. **Persistent State**: State is saved to `~/.config/rocha/state.json` so the status bar works even when the TUI is closed
4. **Execution Filtering**: Only sessions from the current TUI run are counted (prevents stale data from previous runs)

### State Transitions

| Event | State Change | Sound | Meaning |
|-------|-------------|-------|---------|
| `stop` | → waiting | ✓ Yes | Claude finished, needs your input |
| `notification` | → waiting | ✓ Yes | Claude sent a notification (e.g., AskUserQuestion) |
| `prompt` | → working | ✗ No | You submitted a prompt, Claude is processing |
| `start` | → working | ✓ Yes | Session initialized and ready |

## How It Works

This application demonstrates the core pattern used by [claude-squad](https://github.com/smtg-ai/claude-squad):

1. **Bubble Tea TUI**: Uses Charm's Bubble Tea framework for the list interface
2. **tea.ExecProcess**: Suspends the Bubble Tea program and gives tmux full terminal control
3. **Tmux Sessions**: Sessions are created with `tmux new-session -d` (detached)
4. **Native Detach**: Uses tmux's built-in detach (Ctrl+B D) to return to the list
5. **Claude Code Hooks**: Integrates with Claude's hook system for notifications

### Key Technical Details

- Tmux sessions are created with `tmux new-session -d` (detached)
- `tea.ExecProcess` suspends Bubble Tea and runs `tmux attach-session` with full terminal control
- When you detach from tmux (Ctrl+B D), control returns to the Bubble Tea list view
- The list view is rendered using Bubble Tea's Elm-like architecture
- Session list is refreshed after detaching to show current state

### Notification System

When you create a new session, rocha automatically:

1. Starts Claude with `rocha start-claude` instead of plain `claude`
2. Configures four Claude Code hooks via `--settings` flag:
   - **Stop hook**: Fires when Claude finishes and needs input
   - **Notification hook**: Fires when Claude sends notifications (e.g., AskUserQuestion)
   - **UserPromptSubmit hook**: Fires when you submit a prompt
   - **SessionStart hook**: Fires when the session initializes
3. Each hook calls `rocha notify <session-name> <event-type>`
4. Cross-platform sound playing:
   - **macOS**: Uses `afplay` with system sounds (Glass, Ping, Submarine)
   - **Linux**: Uses `paplay`/`aplay` with freedesktop sounds
   - **Windows**: Uses PowerShell SystemSounds
   - **Fallback**: Terminal bell (`\a`) if audio unavailable

The hooks configuration is injected as inline JSON, avoiding the need for settings files.

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework with process execution
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [google/uuid](https://github.com/google/uuid) - UUID generation for execution tracking
- [stretchr/testify](https://github.com/stretchr/testify) - Testing assertions
- [golang.org/x/sys](https://golang.org/x/sys) - System calls for file locking

## Requirements

- Go 1.16+
- tmux installed and available in PATH
- Claude Code CLI (`claude`) installed and available in PATH

## License

MIT
