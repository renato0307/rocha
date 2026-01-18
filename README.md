# Rocha

I'm Rocha, and I manage coding agents.
In practical terms, I'm a tmux-based session manager for Claude Code CLI.

## Requirements

- tmux
- Claude Code CLI (`claude`)
- git (optional, for worktree support)
- Go 1.23+ (only if building from source)

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
- **Shell sessions** - Open a separate shell (⌨) for each Claude session
- **Rename sessions** - Give your sessions meaningful names
- **Manual ordering** - Organize sessions by moving them up/down
- **Quick attach** - Jump to sessions 1-7 with alt+number keys
- **Editor integration** - Open sessions directly in your editor
- **Filter sessions** - Search sessions by name or git branch
- **Get sound alerts** - Hear when Claude finishes and needs your input
- **See status in tmux** - Show active/waiting sessions in your status bar
- **Session states** - Track which sessions are working, idle, waiting, or exited
- **Git worktree support** - Each session can have its own isolated branch and workspace
- **Git stats** - See PR info, ahead/behind commits, and changes at a glance

## Key Bindings

**In the list:**
- `n` - New session
- `Enter` - Attach to session
- `alt+1` to `alt+7` - Quick attach to sessions by number
- `alt+enter` - Open shell session (⌨) for the selected session
- `r` - Rename session
- `c` - Add/edit comment on session
- `f` - Toggle flag (mark session as important)
- `s` - Cycle through statuses quickly (working → idle → waiting → exited)
- `shift+s` - Set status (interactive form to choose status)
- `o` - Open session in your editor
- `x` - Kill session
- `↑/↓` or `j/k` - Navigate
- `shift+↑/↓` or `shift+j/k` - Move session up/down in list
- `/` - Filter/search sessions
- `esc` (twice) - Clear filter
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

The `rocha setup` command adds session counts to your tmux status bar.

You'll see `W:2 A:1` meaning 2 sessions waiting for input, 1 actively working.

Or add manually to `~/.tmux.conf`:
```bash
set -g status-right "Claude: #(rocha status) | %H:%M"
set -g status-interval 1
```

## Building from Source

```bash
make install        # Build and install to ~/.local/bin
make build          # Build only
go build -o rocha . # Build with Go directly
```

Check your installation:
```bash
rocha --version
```

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

## Release Process (Maintainers)

Create and push a version tag to trigger automated release:
```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

Test locally first:
```bash
make snapshot
./dist/rocha_linux_amd64_v1/rocha --version
```
