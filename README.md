# Rocha

I'm Rocha, and I manage coding agents

## Quick Start

```bash
# Build and install
make install

# Set up PATH and tmux status bar
rocha setup
source ~/.zshrc  # or ~/.bashrc

# Run it
rocha
```

Press `n` to create your first session, `Enter` to attach, `Ctrl+Q` to return to the list.

For all available commands and options, run `rocha --help`.

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

When running in a git repository, `rocha` offers to create isolated worktrees for each session:

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
set -g mouse on  # Enable mouse support
```

## Building from Source

### Prerequisites
- Go 1.25.6 or later
- Git

### Using Make (recommended)
```bash
make build          # Build with version info
make install        # Build and install to ~/.local/bin
make snapshot       # Test release build locally
```

### Using Go directly
```bash
go build -o rocha .
```

### Check Version
```bash
rocha --version
```

## Release Process (for maintainers)

Releases are automated via GitHub Actions:

1. Create and push a version tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. GitHub Actions automatically:
   - Builds binaries for linux/darwin on amd64/arm64
   - Creates a GitHub release with changelog
   - Uploads binaries and checksums

3. Test locally before tagging:
   ```bash
   make snapshot
   ./dist/rocha_linux_amd64_v1/rocha --version
   ```

## Requirements

- tmux
- git (for worktree support)
- Claude Code CLI (`claude`)
- Go 1.25.6+ (for building from source)

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
