# Rocha Architecture

TUI application for managing Claude Code sessions via tmux with git worktree support.

## System Overview

```mermaid
graph TB
    subgraph "CLI Layer"
        CLI[CLI Commands<br/>cmd/]
    end

    subgraph "UI Layer"
        TUI[TUI<br/>ui/]
    end

    subgraph "Business Logic"
        TMUX[Tmux Client<br/>tmux/]
        STATE[State Manager<br/>state/]
        GIT[Git Operations<br/>git/]
        EDITOR[Editor Opener<br/>editor/]
    end

    subgraph "Cross-Cutting Concerns"
        LOG[Logging<br/>logging/]
        VER[Version<br/>version/]
    end

    subgraph "External"
        TMUXCLI[tmux CLI]
        GITCLI[git CLI]
        FS[File System<br/>state.json]
        LOGFS[Log Files]
    end

    CLI --> TUI
    CLI --> TMUX
    CLI --> GIT
    CLI --> STATE
    TUI --> TMUX
    TUI --> STATE
    TUI --> GIT
    TUI --> EDITOR
    TMUX --> TMUXCLI
    STATE --> FS
    GIT --> GITCLI

    %% Cross-cutting concerns (dotted lines)
    CLI -.-> LOG
    TUI -.-> LOG
    TUI -.-> VER
    TMUX -.-> LOG
    LOG --> LOGFS
```

### Cross-Cutting Concerns

Components shown with dotted lines (-.->) are **cross-cutting concerns** - they're used across multiple layers but don't participate in the main architectural flow:

- **logging/**: Structured logging (slog) used by cmd, ui, and tmux for debugging and audit trails
  - No business logic depends on logging
  - Can be disabled/redirected without affecting core functionality
  - Used for: operation traces, error diagnostics, debugging

- **version/**: Version and tagline constants
  - Read-only data used for display
  - No behavioral dependencies

These packages are designed to be:
- **Non-invasive**: Removing them doesn't break business logic
- **Uni-directional**: They don't call back into application code
- **Replaceable**: Can swap implementations (e.g., different log backends)

## Component Architecture

```mermaid
classDiagram
    class Client {
        <<interface>>
        SessionManager
        Attacher
        PaneOperations
        Configurator
    }

    class DefaultClient {
        +Create()
        +List()
        +Kill()
        +Attach()
        +SendKeys()
    }

    class StateManager {
        +AddSession()
        +RemoveSession()
        +Sync()
        +Load()
        +GetCounts()
    }

    class Git {
        +CreateWorktree()
        +RemoveWorktree()
        +IsGitRepo()
        +GetRepoInfo()
    }

    class Model {
        +Update()
        +View()
        +sessionList: SessionList
        +sessionForm: SessionForm
    }

    class SessionList {
        +Init() "Start polling"
        +Update()
        +View()
        +RefreshFromState()
        -pollStateCmd() "Every 2s"
    }

    class SessionForm {
        +Init()
        +Update()
        +View()
        +Result()
        +createSessionCmd() "Async creation"
    }

    DefaultClient ..|> Client
    Model --> Client
    Model --> StateManager
    Model --> Git
    Model --> SessionList
    Model --> SessionForm
    SessionList --> StateManager
    SessionList --> Client
    SessionForm --> StateManager
    SessionForm --> Client

    note for SessionList "Auto-refreshes every 2s\nby checking state.json\ntimestamp (UpdatedAt)"
```

## Data Flow

### Session Creation Flow

```mermaid
sequenceDiagram
    participant User
    participant TUI
    participant Cmd as tea.Cmd<br/>(Background)
    participant Tmux
    participant State
    participant Git

    User->>TUI: Create session
    TUI->>Cmd: createSessionCmd()
    Note over TUI: UI remains responsive
    Cmd->>Git: Create worktree
    Cmd->>Tmux: Create tmux session
    Cmd->>State: Save metadata
    State->>State: Write state.json
    Cmd->>TUI: sessionCreatedMsg

    User->>TUI: Attach to session
    TUI->>Tmux: Attach
    Tmux-->>User: Session shell

    User->>Tmux: Detach
    Tmux-->>TUI: Resume
    TUI->>State: Sync sessions
```

### State Update Flow (Auto-refresh)

```mermaid
sequenceDiagram
    participant Claude
    participant Hook
    participant State
    participant TUI

    Note over Claude,TUI: Session starts
    Claude->>Hook: SessionStart event
    Hook->>State: rocha notify start
    State->>State: Set StateIdle (○)

    loop Every 2 seconds
        TUI->>State: Check state.json timestamp
        State-->>TUI: UpdatedAt changed
        TUI->>TUI: Reload & display ○ (yellow)
    end

    Note over Claude,TUI: User submits prompt
    Claude->>Hook: UserPromptSubmit event
    Hook->>State: rocha notify prompt
    State->>State: Set StateWorking (●)
    TUI->>State: Poll detects change
    TUI->>TUI: Display ● (green)

    Note over Claude,TUI: Claude finishes task
    Claude->>Hook: Stop event
    Hook->>State: rocha notify stop
    State->>State: Set StateIdle (○)
    TUI->>State: Poll detects change
    TUI->>TUI: Display ○ (yellow)

    Note over Claude,TUI: Claude needs input
    Claude->>Hook: Notification event
    Hook->>State: rocha notify notification
    State->>State: Set StateWaitingUser (◐)
    TUI->>State: Poll detects change
    TUI->>TUI: Display ◐ (red)

    Note over Claude,TUI: Claude exits
    Claude->>Hook: SessionEnd event
    Hook->>State: rocha notify end
    State->>State: Set StateExited (■)
    TUI->>State: Poll detects change
    TUI->>TUI: Display ■ (gray)
```

### Hook Event Mapping

Claude Code hooks trigger state transitions:

| Hook Event | Command | New State | Symbol | Meaning |
|------------|---------|-----------|--------|---------|
| `SessionStart` | `rocha notify start` | `StateIdle` | ○ (yellow) | Session initialized and ready |
| `UserPromptSubmit` | `rocha notify prompt` | `StateWorking` | ● (green) | User submitted prompt |
| `Stop` | `rocha notify stop` | `StateIdle` | ○ (yellow) | Claude finished working |
| `Notification` | `rocha notify notification` | `StateWaitingUser` | ◐ (red) | Claude needs user input |
| `SessionEnd` | `rocha notify end` | `StateExited` | ■ (gray) | Claude has exited |

## Packages

### Core Application Packages

#### cmd/
CLI command handlers using Kong framework.
- `RunCmd` - Start TUI
- `AttachCmd` - Register session (creates tmux session, updates state)
- `StatusCmd` - Status bar (`◐:N ○:N ●:N ■:N`)
- `SetupCmd` - Shell integration
- `StartClaudeCmd` - Bootstrap Claude Code with hooks (hidden)
- `NotifyCmd` - Handle Claude hook events (hidden)
- `PlaySoundCmd` - Play notification sounds (hidden)

#### ui/
Bubble Tea TUI with component architecture.
- **SessionList** - Uses bubbles/list component, auto-refresh (2s poll), fuzzy filtering, status symbols (●○◐■)
- **SessionForm** - Session creation forms
- **Model** - Orchestrates states (list, creating, confirming)

#### tmux/
Tmux abstraction layer with dependency injection.
- `Client` interface - Tmux operations
- `DefaultClient` - Implementation via tmux CLI
- `Monitor` - Background session monitoring

#### state/
Persistent session state management.
- JSON storage with file locking
- Session metadata (git info, status, timestamps)
- Four states: `WaitingUser` (◐), `Idle` (○), `Working` (●), `Exited` (■)

#### git/
Git worktree operations.
- Create/remove worktrees
- Branch detection
- Repository metadata extraction

#### editor/
Editor integration with platform-specific defaults.
- Build-tag based platform detection (linux, darwin, windows)
- Fallback chain: CLI flag → `$ROCHA_EDITOR` → `$VISUAL` → `$EDITOR` → platform defaults
- Non-blocking editor launch

### Cross-Cutting Concerns

#### logging/
Structured logging with slog (used by cmd, ui, tmux).
- OS-specific log directories
- JSON log format
- Non-blocking operation tracing

#### version/
Version and build metadata (used by cmd, ui).
- Build-time version injection (Version, Commit, Date, GoVersion)
- Application tagline for display
- Injectable via ldflags during compilation

## Dependencies

```mermaid
graph LR
    subgraph "Go Libraries"
        KONG[kong - CLI]
        BT[bubbletea - TUI]
        BUBBLES[bubbles - Components]
        HUH[huh - Forms]
        LG[lipgloss - Styles]
    end

    subgraph "System Tools"
        TMUX[tmux]
        GIT[git]
        CLAUDE[claude<br/>Claude Code CLI]
        CODE[code<br/>VS Code/Editor]
    end

    APP[Rocha] --> KONG
    APP --> BT
    APP --> BUBBLES
    APP --> HUH
    APP --> LG
    APP --> TMUX
    APP --> GIT
    APP -.-> CLAUDE
    APP -.-> CODE
```

**Go Libraries:**
- `kong` - CLI framework with dependency injection
- `bubbletea` - Terminal UI framework
- `bubbles` - Pre-built TUI components (list, textinput, etc.)
- `huh` - Form components
- `lipgloss` - Styling
- `uuid` - UUID generation
- `unix` - File locking

**External Tools:**
- `tmux` - Terminal multiplexer (required)
- `git` - Version control (required for worktrees)
- `claude` - Claude Code CLI (bootstrapped automatically, dotted line indicates it's spawned not directly called)
- `code` - VS Code or other editor (optional, dotted line indicates it's spawned via 'o' key, falls back to shell)


<!-- Keep this document more visual than textual, an image is better than 1000 words -->