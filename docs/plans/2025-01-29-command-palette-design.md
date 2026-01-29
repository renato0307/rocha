# Command Palette Design

## Overview

Add a command palette to Rocha, similar to VS Code's command palette, providing quick access to all available actions through a searchable overlay.

## Requirements

1. **Shortcut:** `Shift+O` to open, `Esc` to close
2. **Context-aware:** Only show actions valid for current state (e.g., hide session-scoped actions when no session selected)
3. **Flat list, alphabetical ordering**
4. **Fuzzy matching** on action name and description
5. **Dimmed background** when palette is active (new visual pattern)
6. **Immediate execution** on selection (existing confirmations still apply for destructive actions)

## Architecture

### Domain Layer: Action Registry

**New file:** `internal/domain/actions.go`

Introduces `Action` as a domain concept - the canonical source of truth for what actions exist in the system.

```go
package domain

type Action struct {
    Description     string
    Name            string
    RequiresSession bool
}

var Actions = []Action{
    {Name: "archive", Description: "Hide session from the list", RequiresSession: true},
    {Name: "comment", Description: "Add or edit session comment", RequiresSession: true},
    {Name: "cycle_status", Description: "Cycle through implementation statuses", RequiresSession: true},
    {Name: "flag", Description: "Toggle session flag", RequiresSession: true},
    {Name: "help", Description: "Show keyboard shortcuts", RequiresSession: false},
    {Name: "kill", Description: "Kill session and optionally remove worktree", RequiresSession: true},
    {Name: "new_from_repo", Description: "Create session from same repository", RequiresSession: true},
    {Name: "new_session", Description: "Create a new session", RequiresSession: false},
    {Name: "open", Description: "Attach to Claude session", RequiresSession: true},
    {Name: "open_editor", Description: "Open session folder in editor", RequiresSession: true},
    {Name: "open_shell", Description: "Open or attach to shell session", RequiresSession: true},
    {Name: "quit", Description: "Exit Rocha", RequiresSession: false},
    {Name: "rename", Description: "Rename session", RequiresSession: true},
    {Name: "send_text", Description: "Send text to session (experimental)", RequiresSession: true},
    {Name: "set_status", Description: "Choose implementation status", RequiresSession: true},
    {Name: "timestamps", Description: "Toggle timestamp display", RequiresSession: false},
    {Name: "token_chart", Description: "Toggle token usage chart", RequiresSession: false},
}

func GetActions() []Action {
    return Actions
}

func GetActionByName(name string) *Action {
    for _, a := range Actions {
        if a.Name == name {
            return &a
        }
    }
    return nil
}
```

### UI Layer: Command Palette Component

**New file:** `internal/ui/command_palette.go`

```go
type CommandPalette struct {
    actions        []domain.Action  // Filtered list based on context
    allActions     []domain.Action  // Complete list from domain
    filterInput    textinput.Model  // Fuzzy search input
    height         int
    selectedIndex  int
    selectedItem   *SessionItem     // nil if no session selected
    width          int
}
```

**Behavior:**
- `Init()` - Focuses the filter input
- `Update()` - Handles `Esc` (close), `Enter` (execute), `Up/Down` (navigate), typing (filter)
- `View()` - Renders the palette with header showing selected session name (if any)

**Fuzzy filtering:**
- Matches if all characters appear in order in either name or description
- Case-insensitive
- Re-filters on every keystroke

### UI Layer: Action Dispatcher

**New file:** `internal/ui/action_dispatcher.go`

Maps domain action names to UI messages. This keeps the command palette decoupled from specific message types.

```go
type ActionDispatcher struct {
    selectedSession *SessionItem
}

func (d *ActionDispatcher) Dispatch(action domain.Action) tea.Msg {
    switch action.Name {
    case "archive":
        return ArchiveSessionMsg{SessionName: d.selectedSession.Session.Name}
    case "new_session":
        return NewSessionMsg{}
    // ... etc
    }
    return nil
}
```

### UI Layer: Dimmed Background Overlay

New rendering pattern in `model.go` for `stateCommandPalette`:

1. Render normal list view as background
2. Apply dim/muted style to each line
3. Render palette centered on top
4. Composite using lipgloss positioning

### Key Binding Integration

**Changes to `keys_definitions.go`:**
```go
{Name: "command_palette", Defaults: []string{"O"}, Help: "command palette", TipFormat: "press %s to open the command palette"},
```

**Changes to `keys_application.go`:**
Add `CommandPalette KeyWithTip` to `ApplicationKeys` struct.

**Changes to `session_list.go`:**
Handle `Shift+O` to emit `ShowCommandPaletteMsg`.

**Changes to `model.go`:**
- Add `stateCommandPalette` to state constants
- Add `commandPalette *CommandPalette` field
- Handle `ShowCommandPaletteMsg` to transition state
- Handle `ExecuteActionMsg` to dispatch via `ActionDispatcher`
- Update `View()` to render dimmed overlay when in palette state

**Changes to `messages.go`:**
```go
type ShowCommandPaletteMsg struct{}

type ExecuteActionMsg struct {
    ActionName string
}

type CloseCommandPaletteMsg struct{}
```

## File Changes Summary

### New Files
- `internal/domain/actions.go` - Action definitions
- `internal/ui/command_palette.go` - Palette component
- `internal/ui/action_dispatcher.go` - Action-to-message mapping

### Modified Files
- `internal/ui/keys_definitions.go` - Add command_palette key
- `internal/ui/keys_application.go` - Add CommandPalette to ApplicationKeys
- `internal/ui/session_list.go` - Handle Shift+O
- `internal/ui/model.go` - State management, dimmed overlay rendering
- `internal/ui/messages.go` - New message types

## Technical Debt

**Issue to create:** The introduction of `domain.Actions` as the source of truth for actions should be leveraged by the entire system. Currently:

- Help screen reads from `KeyMap` → `KeyDefinitions`
- Tips read from `KeyDefinitions.TipFormat`
- Command palette will read from `domain.Actions`

**Follow-up work:** Refactor help screen and tips to derive from `domain.Actions` linked with `KeyDefinitions`, eliminating the inconsistency.

## Visual Reference

The command palette should appear as shown in the mockup:
- Header with selected session name (if applicable)
- Two-column layout: action name (left), description (right)
- Selected item highlighted
- Filter input at bottom with "Type to filter..." placeholder
- Footer with navigation hints: "↑↓ navigate • ⏎ select • esc cancel"
- Background dimmed behind the palette
