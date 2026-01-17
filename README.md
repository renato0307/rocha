# Rocha

Manage multiple Claude Code sessions in your terminal. Switch between conversations like browser tabs, with sound notifications when Claude needs your attention.

## Quick Start

```bash
# Build and install
go build -o rocha
cp rocha ~/.local/bin/

# Run it
rocha
```

Press `n` to create your first session, `Enter` to attach, `Ctrl+Q` to return to the list.

## What You Can Do

- **Switch between Claude sessions** - Keep multiple conversations organized
- **Get sound alerts** - Hear when Claude finishes and needs your input
- **See status in tmux** - Show active/waiting sessions in your status bar
- **Never lose context** - Sessions persist until you kill them
- **Git worktree support** - Each session can have its own isolated branch and workspace

## Key Bindings

**In the list:**
- `n` - New session
- `Enter` - Attach to session
- `x` - Kill session
- `↑/↓` or `j/k` - Navigate
- `q` - Quit

**Inside a session:**
- `Ctrl+Q` - Quick return to list
- `Ctrl+B then D` - Standard tmux detach (also works)

## Git Worktree Support

When running in a git repository, rocha offers to create isolated worktrees for each session:

- **Automatic detection** - Detects git repos and prompts for worktree creation
- **Isolated branches** - Each session gets its own branch and working directory
- **No conflicts** - Work on multiple branches simultaneously without switching
- **Auto cleanup** - Worktrees are removed when you kill the session

Worktrees are stored in `~/.rocha/worktrees/` by default.

## Status Bar (Optional)

Show session counts in your tmux status bar:

```bash
rocha setup
source ~/.zshrc  # or ~/.bashrc
```

You'll see `W:2 A:1` meaning 2 sessions waiting for input, 1 actively working.

Or add manually to `~/.tmux.conf`:
```bash
set -g status-right "Claude: #(rocha status) | %H:%M"
set -g status-interval 1
```

## Requirements

- tmux
- Claude Code CLI (`claude`)
- Go 1.16+ (for building)

## Troubleshooting

```bash
rocha --debug              # Enable logging
rocha --help               # More options
```

Logs go to:
- Linux: `~/.local/state/rocha/`
- macOS: `~/Library/Logs/rocha/`
- Windows: `%LOCALAPPDATA%\rocha\logs\`

## License

GPL v3
