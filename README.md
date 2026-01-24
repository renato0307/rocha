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

## Configuration

### ROCHA_HOME

Set `ROCHA_HOME` to customize where rocha stores data:

```bash
export ROCHA_HOME=~/my-custom-rocha
```

**Default:** `~/.rocha`

This directory contains:
- `state.db` - Session database
- `worktrees/` - Git worktrees for sessions
- `settings.json` - Configuration settings

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
- **Per-session Claude config** - Give each session its own Claude configuration directory
- **Create sessions from any repo** - Clone and start sessions from GitHub/GitLab URLs with specific branches

## Session States

Rocha automatically tracks the state of each Claude session using hooks:

### State Symbols

- **● (green)** - **Working**: Claude is actively processing a task
- **○ (yellow)** - **Idle**: Claude finished its turn, ready for your next prompt
- **◐ (red)** - **Waiting**: Claude is blocked on a UI interaction (form, permission dialog)
- **■ (gray)** - **Exited**: Claude has exited the session

### State Transitions

```
User submits prompt → working (●)
Claude finishes task → idle (○)
Claude shows AskUserQuestion form → waiting (◐)
User answers form → working (●)
Claude needs permission → waiting (◐)
Claude exits → exited (■)
```

### Key Differences

**idle (○) vs waiting (◐)**:
- **idle**: Claude finished, you type your next message in chat
- **waiting**: Claude is blocked on a UI element (form with buttons, permission dialog)

For example:
- Claude asks "What color?" in text → **idle (○)** (normal chat)
- Claude shows `[○ Red] [○ Blue]` form → **waiting (◐)** (blocking UI)

## Key Bindings

**In the list:**
- `n` - new session
- `enter` - attach to session
- `alt+1` to `alt+7` - quick attach to sessions by number
- `alt+enter` - open shell session (⌨) for the selected session
- `r` - rename session
- `c` - add/edit comment on session
- `f` - toggle flag (mark session as important)
- `s` - cycle through statuses quickly (working → idle → waiting → exited)
- `shift+s` - set status (interactive form to choose status)
- `o` - open session in your editor
- `x` - kill session
- `↑/↓` or `j/k` - navigate
- `shift+↑/↓` or `shift+j/k` - move session up/down in list
- `/` - filter/search sessions
- `esc` (twice) - clear filter
- `q` - quit

**Inside a session:**
- `ctrl+q` - quick return to list
- `ctrl+]` - swap between claude and shell sessions
- `ctrl+b then d` - standard tmux detach (also works)

## Git Worktree Support

When running in a git repository, `rocha` offers to create isolated worktrees for each session:

- **Automatic detection** - Detects git repos and prompts for worktree creation
- **Isolated branches** - Each session gets its own branch and working directory
- **No conflicts** - Work on multiple branches simultaneously without switching
- **Auto cleanup** - Worktrees are removed when you kill the session

Worktrees are stored in `$ROCHA_HOME/worktrees/` (default: `~/.rocha/worktrees/`).

## Creating Sessions from Any Repository

You can create sessions from any git repository (GitHub, GitLab, etc.) without needing to clone it first:

```bash
# Create a session from a specific branch
# In the session form, enter:
https://github.com/owner/repo#branch-name

# Or use the default branch
https://github.com/owner/repo

# You can also use SSH URLs
git@github.com:owner/repo.git#develop
```

**How it works:**
1. Rocha clones the repository to `$ROCHA_HOME/worktrees/owner/repo/.main/` (default: `~/.rocha/worktrees/owner/repo/.main/`)
2. Creates a worktree for your session from the specified branch
3. Multiple sessions from the same repo share the `.main` directory
4. Each session automatically switches `.main` to the correct branch before creating its worktree

**Benefits:**
- Start working on any project instantly without manual cloning
- Work on multiple branches from the same repo simultaneously
- Each session gets the correct base branch automatically
- No branch conflicts between sessions

**Example workflow:**
```bash
# Session 1: Work on the main branch
Repository: https://github.com/myorg/myapp#main

# Session 2: Work on a feature branch (same repo)
Repository: https://github.com/myorg/myapp#feature/new-ui

# Both sessions work independently with correct branches!
```

## Per-Session Claude Configuration

Each session can have its own Claude configuration directory, allowing you to:

- Use different Claude accounts or API keys per session
- Isolate conversation history between projects
- Test different Claude settings without affecting other sessions
- Share Claude config with team members (stored in project directory)

**Default behavior:**
- All sessions use `~/.claude` by default
- Sessions from the same repository automatically share the same Claude directory

**Custom configuration:**
When creating a session, you can specify a custom Claude directory:
```bash
# In the session form, edit the "Claude directory" field:
/path/to/project/.claude

# Or use a shared team config
~/team-configs/project-a/.claude
```

**Environment variable:**
Rocha sets `CLAUDE_CONFIG_DIR` for each session, which Claude Code reads to determine where to store its configuration, history, and cache.

**Use cases:**
1. **Per-project isolation** - Keep each project's Claude conversations separate
2. **Testing** - Use a separate config for experimental sessions
3. **Team collaboration** - Share Claude config files in your git repository
4. **Multiple accounts** - Switch between different Claude accounts easily

**Example:**
```bash
# Work session using work Claude account
Claude directory: ~/.claude-work

# Personal project using personal account
Claude directory: ~/.claude-personal

# Project-specific (committed to git)
Claude directory: /path/to/project/.claude
```

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

## Testing

```bash
make test                              # Run all tests
make test-integration                  # Run integration tests
make test-integration-verbose          # Run with no cache
make test-integration-run TEST=TestName  # Run specific test
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
