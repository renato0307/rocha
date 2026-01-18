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
        STORAGE[Storage Layer<br/>storage/]
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
        DB[SQLite Database<br/>rocha.db]
        LOGFS[Log Files]
    end

    CLI --> TUI
    CLI --> TMUX
    CLI --> GIT
    CLI --> STORAGE
    TUI --> TMUX
    TUI --> STORAGE
    TUI --> GIT
    TUI --> EDITOR
    TMUX --> TMUXCLI
    STORAGE --> DB
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

    class Store {
        +Load()
        +Save()
        +UpdateSession()
        +UpdateSessionStatus()
        +SwapPositions()
        +ToggleFlag()
        +GetSession()
        +ListSessions()
    }

    class StatusConfig {
        +Statuses: []string
        +Colors: []string
        +GetColor()
        +GetNextStatus()
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
        +sessionStatusForm: SessionStatusForm
        +statusConfig: StatusConfig
    }

    class SessionList {
        +Init() "Start polling"
        +Update()
        +View()
        +RefreshFromState()
        +cycleSessionStatus()
        -pollStateCmd() "Every 2s"
    }

    class SessionForm {
        +Init()
        +Update()
        +View()
        +Result()
        +createSessionCmd() "Async creation"
    }

    class SessionStatusForm {
        +Init()
        +Update()
        +View()
        +Result()
        +updateStatus() "Update status in DB"
    }

    DefaultClient ..|> Client
    Model --> Client
    Model --> Store
    Model --> Git
    Model --> StatusConfig
    Model --> SessionList
    Model --> SessionForm
    Model --> SessionStatusForm
    SessionList --> Store
    SessionList --> Client
    SessionList --> StatusConfig
    SessionForm --> Store
    SessionForm --> Client
    SessionStatusForm --> Store
    SessionStatusForm --> StatusConfig

    note for SessionList "Auto-refreshes every 2s\nby checking database\nupdates (UpdatedAt)"
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
    participant Storage
    participant TUI

    Note over Claude,TUI: Session starts
    Claude->>Hook: SessionStart event
    Hook->>Storage: rocha notify start
    Storage->>Storage: Update StateIdle (○) in DB

    loop Every 2 seconds
        TUI->>Storage: Load() from database
        Storage-->>TUI: Session data with UpdatedAt
        TUI->>TUI: Reload & display ○ (yellow)
    end

    Note over Claude,TUI: User submits prompt
    Claude->>Hook: UserPromptSubmit event
    Hook->>Storage: rocha notify prompt
    Storage->>Storage: Update StateWorking (●) in DB
    TUI->>Storage: Poll detects change
    TUI->>TUI: Display ● (green)

    Note over Claude,TUI: Claude finishes task
    Claude->>Hook: Stop event
    Hook->>Storage: rocha notify stop
    Storage->>Storage: Update StateIdle (○) in DB
    TUI->>Storage: Poll detects change
    TUI->>TUI: Display ○ (yellow)

    Note over Claude,TUI: Claude needs input
    Claude->>Hook: Notification event
    Hook->>Storage: rocha notify notification
    Storage->>Storage: Update StateWaitingUser (◐) in DB
    TUI->>Storage: Poll detects change
    TUI->>TUI: Display ◐ (red)

    Note over Claude,TUI: Claude exits
    Claude->>Hook: SessionEnd event
    Hook->>Storage: rocha notify end
    Storage->>Storage: Update StateExited (■) in DB
    TUI->>Storage: Poll detects change
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

### Session Status Update Flow

```mermaid
sequenceDiagram
    participant User
    participant TUI
    participant Storage
    participant DB

    Note over User,DB: Option 1: Form-based status selection
    User->>TUI: Press 's' on session
    TUI->>TUI: Show SessionStatusForm
    User->>TUI: Select status or <clear>
    TUI->>Storage: UpdateSessionStatus()
    Storage->>DB: Upsert/Delete in session_statuses table
    Storage-->>TUI: Success
    TUI->>Storage: Load() to refresh
    TUI->>TUI: Display colored status [status]

    Note over User,DB: Option 2: Quick status cycling
    User->>TUI: Press Shift+S on session
    TUI->>TUI: cycleSessionStatus()
    TUI->>Storage: UpdateSessionStatus(next_status)
    Storage->>DB: Upsert/Delete in session_statuses table
    Storage-->>TUI: Success
    TUI->>TUI: Immediately refresh (checkStateMsg)
    TUI->>TUI: Display colored status [status]
```

#### Status Feature Details

- **Storage**: Status stored in separate `session_statuses` table (1-to-1 with sessions)
- **Customization**:
  - `--statuses` flag: Comma-separated status names (default: "spec,plan,implement,review,done")
  - `--status-colors` flag: ANSI color codes for each status (default: "141,33,214,226,46")
- **Display**: Status shown in brackets next to session name with color coding
- **Keyboard shortcuts**:
  - `s`: Open status selection form with `<clear>` option
  - `Shift+S`: Cycle through statuses (nil → first → ... → last → nil)
- **Restrictions**: Only top-level sessions can have status (nested/shell sessions excluded)

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
- **SessionList** - Uses bubbles/list component, auto-refresh (2s poll), fuzzy filtering, status symbols (●○◐■), status display with colors
- **SessionForm** - Session creation forms
- **SessionStatusForm** - Status selection form with `<clear>` option
- **StatusConfig** - Configuration for status names, colors, and cycling logic
- **Model** - Orchestrates states (list, creating, confirming, settingStatus)

#### tmux/
Tmux abstraction layer with dependency injection.
- `Client` interface - Tmux operations
- `DefaultClient` - Implementation via tmux CLI
- `Monitor` - Background session monitoring

#### storage/
Persistent session storage using SQLite.
- **Store** - GORM-based database abstraction with ACID guarantees
- **Schema** - Session, SessionFlag, SessionStatus tables
- **Types** - SessionInfo and SessionState for business logic
- SQLite with WAL mode for concurrent access
- Atomic transactions with retry logic for busy database scenarios
- Session metadata (git info, status, timestamps, flags)
- Four session states: `WaitingUser` (◐), `Idle` (○), `Working` (●), `Exited` (■)
- Separate tables for:
  - **sessions** - Core session data with position ordering
  - **session_flags** - Flag/marker for important sessions
  - **session_statuses** - Implementation status tracking (1-to-1 with sessions)

#### git/
Git worktree operations.
- Create/remove worktrees
- Branch name validation and sanitization
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
        GORM[gorm - ORM]
        SQLITE[go-sqlite3 - Driver]
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
    APP --> GORM
    APP --> SQLITE
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
- `gorm` - ORM for database operations with transaction support
- `go-sqlite3` - SQLite driver for Go (CGo-based)
- `uuid` - UUID generation

**External Tools:**
- `tmux` - Terminal multiplexer (required)
- `git` - Version control (required for worktrees)
- `claude` - Claude Code CLI (bootstrapped automatically, dotted line indicates it's spawned not directly called)
- `code` - VS Code or other editor (optional, dotted line indicates it's spawned via 'o' key, falls back to shell)


<!-- Keep this document more visual than textual, an image is better than 1000 words -->