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
        LOG[Logging<br/>logging/]
    end

    subgraph "External"
        TMUXCLI[tmux CLI]
        GITCLI[git CLI]
        FS[File System<br/>state.json]
    end

    CLI --> TUI
    CLI --> TMUX
    TUI --> TMUX
    TUI --> STATE
    TMUX --> TMUXCLI
    TMUX --> GIT
    STATE --> FS
    GIT --> GITCLI
```

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
    }

    class Model {
        +Update()
        +View()
    }

    DefaultClient ..|> Client
    Model --> Client
    Model --> StateManager
```

## Data Flow

```mermaid
sequenceDiagram
    participant User
    participant TUI
    participant Tmux
    participant State
    participant Git

    User->>TUI: Create session
    TUI->>Git: Create worktree
    TUI->>Tmux: Create tmux session
    TUI->>State: Save metadata
    State->>State: Write state.json

    User->>TUI: Attach to session
    TUI->>Tmux: Attach
    Tmux-->>User: Session shell

    User->>Tmux: Detach
    Tmux-->>TUI: Resume
    TUI->>State: Sync sessions
```

## Packages

### cmd/
CLI command handlers using Kong framework.
- `RunCmd` - Start TUI
- `AttachCmd` - Attach to session
- `StatusCmd` - Status bar integration
- `SetupCmd` - Shell integration

### ui/
Bubble Tea TUI implementation.
- Session list view
- Session creation forms
- State machine (list, creating, confirming, filtering)

### tmux/
Tmux abstraction layer with dependency injection.
- `Client` interface - Tmux operations
- `DefaultClient` - Implementation via tmux CLI
- `Monitor` - Background session monitoring

### state/
Persistent session state management.
- JSON storage with file locking
- Session metadata (git info, status, timestamps)
- Sync with tmux sessions

### git/
Git worktree operations.
- Create/remove worktrees
- Branch detection
- Repository metadata extraction

### logging/
Structured logging with slog.
- OS-specific log directories
- JSON log format

### version/
Version and tagline constants.

## Dependencies

```mermaid
graph LR
    subgraph "Go Libraries"
        KONG[kong - CLI]
        BT[bubbletea - TUI]
        HUH[huh - Forms]
        LG[lipgloss - Styles]
    end

    subgraph "System"
        TMUX[tmux]
        GIT[git]
    end

    APP[Rocha] --> KONG
    APP --> BT
    APP --> HUH
    APP --> LG
    APP --> TMUX
    APP --> GIT
```

**Go Libraries:**
- `kong` - CLI framework with dependency injection
- `bubbletea` - Terminal UI framework
- `huh` - Form components
- `lipgloss` - Styling
- `uuid` - UUID generation
- `unix` - File locking

**External Tools:**
- `tmux` - Terminal multiplexer
- `git` - Version control
