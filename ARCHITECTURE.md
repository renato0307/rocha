# Rocha Architecture

TUI application for managing Claude Code sessions via tmux with git worktree support.

## Hexagonal Architecture Overview

```mermaid
graph TB
    subgraph "Outer Layer - Drivers"
        CLI[CLI Commands<br/>cmd/]
        TUI[TUI<br/>ui/]
    end

    subgraph "Application Layer"
        SS[SessionService]
        GS[GitService]
        SHS[ShellService]
        STS[SettingsService]
        NS[NotificationService]
        MS[MigrationService]
        TSS[TokenStatsService]
    end

    subgraph "Domain"
        DOM[Domain Entities<br/>domain/]
    end

    subgraph "Ports Layer"
        SR[SessionRepository]
        GR[GitRepository]
        TC[TmuxClient]
        EO[EditorOpener]
        SP[SoundPlayer]
        PI[ProcessInspector]
        TUR[TokenUsageReader]
    end

    subgraph "Adapters Layer"
        SQLITE[SQLite Adapter<br/>storage/]
        GITCLI[Git CLI Adapter<br/>git/]
        TMUXCLI[Tmux CLI Adapter<br/>tmux/]
        EDITOR[Editor Adapter<br/>editor/]
        SOUND[Sound Adapter<br/>sound/]
        PROCESS[Process Adapter<br/>process/]
        CLAUDE[Claude Session Parser<br/>claude/]
    end

    subgraph "External Systems"
        DB[(SQLite DB)]
        GIT[git CLI]
        TMUX[tmux CLI]
        VSCODE[VS Code/Editor]
        AUDIO[Audio System]
        OS[OS Processes]
        JSONL[(Claude Session JSONL)]
    end

    CLI --> SS
    CLI --> GS
    CLI --> SHS
    CLI --> STS
    CLI --> NS
    CLI --> MS
    CLI --> TSS
    TUI --> SS
    TUI --> GS
    TUI --> SHS
    TUI --> TSS

    SS --> SR
    SS --> GR
    SS --> TC
    SS --> PI
    GS --> GR
    SHS --> SR
    SHS --> TC
    SHS --> EO
    NS --> SR
    NS --> SP
    MS --> GR
    MS --> TC
    STS --> SR
    TSS --> TUR

    SR -.-> SQLITE
    GR -.-> GITCLI
    TC -.-> TMUXCLI
    EO -.-> EDITOR
    SP -.-> SOUND
    PI -.-> PROCESS
    TUR -.-> CLAUDE

    SQLITE --> DB
    GITCLI --> GIT
    TMUXCLI --> TMUX
    EDITOR --> VSCODE
    SOUND --> AUDIO
    PROCESS --> OS
    CLAUDE --> JSONL
```

### Architecture Layers

1. **Drivers (Outer Layer)**: CLI commands and TUI - entry points that use services
2. **Application Services**: Business logic orchestration - facade to the domain
3. **Domain**: Core entities and business rules
4. **Ports**: Interface definitions - contracts between layers
5. **Adapters**: Infrastructure implementations - connect to external systems

### Key Principle

- **CLI and UI only depend on Services** - never on Ports or Adapters directly
- **Services orchestrate Ports** - implement business use cases
- **Adapters implement Ports** - provide infrastructure

## Data Flow

### Session Creation Flow

```mermaid
sequenceDiagram
    participant User
    participant TUI
    participant SS as SessionService
    participant GR as GitRepository<br/>(Port)
    participant TC as TmuxClient<br/>(Port)
    participant SR as SessionRepository<br/>(Port)

    User->>TUI: Create session
    TUI->>SS: CreateSession()
    SS->>GR: CreateWorktree()
    SS->>TC: CreateSession()
    SS->>SR: Add(session)
    SS-->>TUI: SessionResult
    TUI-->>User: Session created
```

### State Update Flow (Hook Events)

```mermaid
sequenceDiagram
    participant Claude
    participant Hook
    participant NS as NotificationService
    participant SR as SessionRepository<br/>(Port)
    participant TUI

    Claude->>Hook: SessionStart event
    Hook->>NS: HandleEvent(start)
    NS->>SR: UpdateState(Idle)

    loop Every 2 seconds
        TUI->>SR: LoadState()
        SR-->>TUI: Session data
        TUI->>TUI: Display state
    end

    Claude->>Hook: UserPromptSubmit
    Hook->>NS: HandleEvent(prompt)
    NS->>SR: UpdateState(Working)
```

### Hook Event Mapping

| Hook Event | New State | Symbol | Meaning |
|------------|-----------|--------|---------|
| `SessionStart` | `StateIdle` | ○ (yellow) | Session initialized |
| `UserPromptSubmit` | `StateWorking` | ● (green) | User submitted prompt |
| `PreToolUse` (AskUserQuestion) | `StateWaiting` | ◐ (red) | Claude asking question |
| `PostToolUse` (AskUserQuestion) | `StateWorking` | ● (green) | User answered question |
| `PostToolUseFailure` | `StateWorking` | ● (green) | Tool failed, continuing |
| `PermissionRequest` | `StateWaiting` | ◐ (red) | Needs user permission |
| `SubagentStart` | `StateWorking` | ● (green) | Delegating to subagent |
| `SubagentStop` | `StateWorking` | ● (green) | Subagent completed |
| `PreCompact` | `StateWorking` | ● (green) | Compressing context |
| `Setup` | `StateWorking` | ● (green) | Repository setup |
| `Stop` | `StateIdle` | ○ (yellow) | Claude finished |
| `SessionEnd` | `StateExited` | ■ (gray) | Claude exited |

## Package Structure

```
internal/
├── cmd/           # CLI commands (drivers)
├── ui/            # TUI components (drivers)
├── theme/         # Centralized colors and lipgloss styles
├── services/      # Application services
├── domain/        # Domain entities
├── ports/         # Interface definitions
├── adapters/      # Infrastructure implementations
│   ├── storage/   # SQLite repository
│   ├── git/       # Git CLI operations
│   ├── tmux/      # Tmux CLI operations
│   ├── editor/    # Editor integration
│   ├── sound/     # Sound playback
│   ├── process/   # Process inspection
│   └── claude/    # Claude session file parsing
├── config/        # Configuration and paths
└── logging/       # Structured logging
```

### Services

| Service | Responsibility |
|---------|----------------|
| SessionService | Session lifecycle (create, kill, archive) |
| GitService | Git and worktree operations |
| ShellService | Tmux pane operations, editor, shell sessions |
| SettingsService | Session configuration (claudedir, permissions) |
| NotificationService | Hook event handling, sounds |
| MigrationService | Move sessions between ROCHA_HOME directories |
| TokenStatsService | Parse Claude session files for token usage stats |

### Ports (Interfaces)

| Port | Methods |
|------|---------|
| SessionRepository | Add, Get, List, Delete, Update*, LoadState, SaveState |
| GitRepository | CreateWorktree, RemoveWorktree, IsGitRepo, GetRepoInfo |
| TmuxClient | CreateSession, KillSession, ListSessions, SendKeys |
| EditorOpener | Open |
| SoundPlayer | Play |
| ProcessInspector | GetClaudeSettings |
| TokenUsageReader | GetTodayUsage |

## Dependencies

```mermaid
graph LR
    subgraph "Go Libraries"
        KONG[kong - CLI]
        BT[bubbletea - TUI]
        GORM[gorm - ORM]
    end

    subgraph "External Tools"
        TMUX[tmux]
        GIT[git]
        GH[gh CLI]
        CLAUDE[claude CLI]
    end

    APP[Rocha] --> KONG
    APP --> BT
    APP --> GORM
    APP --> TMUX
    APP --> GIT
    APP -.-> GH
    APP -.-> CLAUDE
```

## Testing

```
test/integration/
├── harness/           # Test utilities
└── *_test.go          # CLI integration tests
```

<!-- Keep this document visual - diagrams over text -->
