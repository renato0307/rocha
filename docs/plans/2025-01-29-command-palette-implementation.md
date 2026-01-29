# Command Palette Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a VS Code-style command palette to Rocha that provides fuzzy-searchable access to all actions.

**Architecture:** Domain layer defines actions as first-class entities. UI layer provides the command palette component with fuzzy filtering, dimmed background overlay, and action-to-message dispatching.

**Tech Stack:** Go, Bubble Tea (TUI framework), lipgloss (styling)

---

## Task 1: Create Domain Action Registry

**Files:**
- Create: `internal/domain/actions.go`

**Step 1: Create the action type and registry**

```go
package domain

// Action represents a user-invocable action in the system.
// This is the domain-level definition of what actions exist.
type Action struct {
	Description     string
	Name            string
	RequiresSession bool
}

// Actions is the canonical registry of all available actions.
// Sorted alphabetically by Name.
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

// GetActions returns all available actions.
func GetActions() []Action {
	return Actions
}

// GetActionByName returns an action by its name, or nil if not found.
func GetActionByName(name string) *Action {
	for i := range Actions {
		if Actions[i].Name == name {
			return &Actions[i]
		}
	}
	return nil
}

// GetActionsForContext returns actions filtered by context.
// If hasSession is false, actions that require a session are excluded.
func GetActionsForContext(hasSession bool) []Action {
	if hasSession {
		return Actions
	}

	var filtered []Action
	for _, a := range Actions {
		if !a.RequiresSession {
			filtered = append(filtered, a)
		}
	}
	return filtered
}
```

**Step 2: Verify the file compiles**

Run: `go build ./internal/domain/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/domain/actions.go
git commit -m "$(cat <<'EOF'
feat(domain): add action registry as source of truth

Introduces Action as a domain concept with:
- Name, Description, RequiresSession fields
- GetActions(), GetActionByName(), GetActionsForContext() helpers

This is the canonical registry of all user-invocable actions.
Tech debt: Help screen and tips should be refactored to use this (#140)
EOF
)"
```

---

## Task 2: Add Command Palette Key Binding

**Files:**
- Modify: `internal/ui/keys_definitions.go`
- Modify: `internal/ui/keys_application.go`

**Step 1: Add key definition**

In `internal/ui/keys_definitions.go`, add to `AllKeyDefinitions` slice (keep alphabetical within Application keys section):

```go
{Name: "command_palette", Defaults: []string{"O"}, Help: "command palette", TipFormat: "press %s to open the command palette"},
```

**Step 2: Add to ApplicationKeys struct**

In `internal/ui/keys_application.go`, add field to `ApplicationKeys` struct (keep alphabetical):

```go
type ApplicationKeys struct {
	CommandPalette KeyWithTip
	ForceQuit      KeyWithTip
	Help           KeyWithTip
	Quit           KeyWithTip
	Timestamps     KeyWithTip
	TokenChart     KeyWithTip
}
```

**Step 3: Add to newApplicationKeys function**

In `internal/ui/keys_application.go`, add to the return statement in `newApplicationKeys`:

```go
return ApplicationKeys{
	CommandPalette: buildBinding("command_palette", defaults, customKeys),
	ForceQuit:      buildBinding("force_quit", defaults, customKeys),
	// ... rest unchanged
}
```

**Step 4: Verify compilation**

Run: `go build ./internal/ui/...`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/ui/keys_definitions.go internal/ui/keys_application.go
git commit -m "$(cat <<'EOF'
feat(ui): add command_palette key binding (Shift+O)

Adds the key binding infrastructure for the command palette.
The actual palette component and handler will be added next.
EOF
)"
```

---

## Task 3: Add UI Message Types

**Files:**
- Modify: `internal/ui/messages.go`

**Step 1: Add message types**

Add at the end of `internal/ui/messages.go`:

```go
// Command palette messages

// ShowCommandPaletteMsg requests showing the command palette
type ShowCommandPaletteMsg struct{}

// ExecuteActionMsg requests executing an action from the command palette
type ExecuteActionMsg struct {
	ActionName string
}

// CloseCommandPaletteMsg requests closing the command palette
type CloseCommandPaletteMsg struct{}
```

**Step 2: Verify compilation**

Run: `go build ./internal/ui/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/ui/messages.go
git commit -m "$(cat <<'EOF'
feat(ui): add command palette message types

Adds ShowCommandPaletteMsg, ExecuteActionMsg, CloseCommandPaletteMsg
for communication between components.
EOF
)"
```

---

## Task 4: Add Theme Styles for Command Palette

**Files:**
- Modify: `internal/theme/colors.go`
- Modify: `internal/theme/styles.go`

**Step 1: Add colors**

In `internal/theme/colors.go`, add to "UI semantic colors" section:

```go
ColorDimmed    Color = "236" // Dark gray - dimmed background
ColorPaletteSelected Color = "62" // Purple - selected item background
```

**Step 2: Add styles**

In `internal/theme/styles.go`, add new section:

```go
// Command palette styles
var (
	PaletteBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	PaletteHeaderStyle = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true).
		Padding(0, 0, 1, 0)

	PaletteItemStyle = lipgloss.NewStyle().
		Foreground(ColorNormal)

	PaletteItemSelectedStyle = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(ColorPaletteSelected).
		Bold(true)

	PaletteDescStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle)

	PaletteDescSelectedStyle = lipgloss.NewStyle().
		Foreground(ColorNormal).
		Background(ColorPaletteSelected)

	PaletteFilterStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle)

	PaletteFooterStyle = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(1, 0, 0, 0)

	DimmedStyle = lipgloss.NewStyle().
		Foreground(ColorDimmed)
)
```

**Step 3: Verify compilation**

Run: `go build ./internal/theme/...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/theme/colors.go internal/theme/styles.go
git commit -m "$(cat <<'EOF'
feat(theme): add command palette styles

Adds styles for:
- Palette border, header, items, footer
- Selected item highlighting
- Dimmed background overlay
EOF
)"
```

---

## Task 5: Create Action Dispatcher

**Files:**
- Create: `internal/ui/action_dispatcher.go`

**Step 1: Create the dispatcher**

```go
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/ports"
)

// ActionDispatcher maps domain actions to UI messages.
// This keeps the command palette decoupled from specific message types.
type ActionDispatcher struct {
	session *ports.TmuxSession
}

// NewActionDispatcher creates a new action dispatcher.
// session can be nil if no session is selected.
func NewActionDispatcher(session *ports.TmuxSession) *ActionDispatcher {
	return &ActionDispatcher{session: session}
}

// Dispatch returns the appropriate tea.Msg for the given action.
// Returns nil if the action cannot be dispatched (e.g., requires session but none selected).
func (d *ActionDispatcher) Dispatch(action domain.Action) tea.Msg {
	// Safety check: action requires session but none selected
	if action.RequiresSession && d.session == nil {
		return nil
	}

	switch action.Name {
	case "archive":
		return ArchiveSessionMsg{SessionName: d.session.Name}
	case "comment":
		return CommentSessionMsg{SessionName: d.session.Name}
	case "cycle_status":
		return CycleStatusMsg{SessionName: d.session.Name}
	case "flag":
		return ToggleFlagSessionMsg{SessionName: d.session.Name}
	case "help":
		return ShowHelpMsg{}
	case "kill":
		return KillSessionMsg{SessionName: d.session.Name}
	case "new_from_repo":
		return NewSessionFromTemplateMsg{TemplateSessionName: d.session.Name}
	case "new_session":
		return NewSessionMsg{}
	case "open":
		return AttachSessionMsg{Session: d.session}
	case "open_editor":
		return OpenEditorSessionMsg{SessionName: d.session.Name}
	case "open_shell":
		return AttachShellSessionMsg{Session: d.session}
	case "quit":
		return QuitMsg{}
	case "rename":
		return RenameSessionMsg{SessionName: d.session.Name}
	case "send_text":
		return SendTextSessionMsg{SessionName: d.session.Name}
	case "set_status":
		return SetStatusSessionMsg{SessionName: d.session.Name}
	case "timestamps":
		return ToggleTimestampsMsg{}
	case "token_chart":
		return ToggleTokenChartMsg{}
	default:
		return nil
	}
}
```

**Step 2: Add missing message types to messages.go**

Add these new message types to `internal/ui/messages.go`:

```go
// CycleStatusMsg requests cycling the status of a session
type CycleStatusMsg struct {
	SessionName string
}

// ToggleTimestampsMsg requests toggling timestamp display
type ToggleTimestampsMsg struct{}

// ToggleTokenChartMsg requests toggling the token chart
type ToggleTokenChartMsg struct{}
```

**Step 3: Verify compilation**

Run: `go build ./internal/ui/...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/ui/action_dispatcher.go internal/ui/messages.go
git commit -m "$(cat <<'EOF'
feat(ui): add action dispatcher for command palette

Maps domain action names to UI message types, keeping the
command palette decoupled from specific message implementations.

Also adds CycleStatusMsg, ToggleTimestampsMsg, ToggleTokenChartMsg.
EOF
)"
```

---

## Task 6: Create Command Palette Component

**Files:**
- Create: `internal/ui/command_palette.go`

**Step 1: Create the component**

```go
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/theme"
)

// CommandPalette is a searchable action palette overlay.
type CommandPalette struct {
	Completed     bool
	Result        CommandPaletteResult
	actions       []domain.Action // Filtered actions
	allActions    []domain.Action // All available actions for context
	filterInput   textinput.Model
	height        int
	selectedIndex int
	session       *ports.TmuxSession // Selected session (can be nil)
	sessionName   string             // Display name for header
	width         int
}

// CommandPaletteResult contains the result of the command palette interaction.
type CommandPaletteResult struct {
	Action    *domain.Action
	Cancelled bool
}

// NewCommandPalette creates a new command palette.
// session can be nil if no session is selected.
// sessionName is the display name to show in the header.
func NewCommandPalette(session *ports.TmuxSession, sessionName string) *CommandPalette {
	hasSession := session != nil
	actions := domain.GetActionsForContext(hasSession)

	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 40

	return &CommandPalette{
		actions:       actions,
		allActions:    actions,
		filterInput:   ti,
		selectedIndex: 0,
		session:       session,
		sessionName:   sessionName,
	}
}

// Init initializes the command palette.
func (cp *CommandPalette) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the command palette.
func (cp *CommandPalette) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cp.width = msg.Width
		cp.height = msg.Height
		return cp, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			cp.Completed = true
			cp.Result.Cancelled = true
			return cp, nil

		case "enter":
			if len(cp.actions) > 0 && cp.selectedIndex < len(cp.actions) {
				cp.Completed = true
				cp.Result.Action = &cp.actions[cp.selectedIndex]
			}
			return cp, nil

		case "up", "ctrl+p":
			if cp.selectedIndex > 0 {
				cp.selectedIndex--
			}
			return cp, nil

		case "down", "ctrl+n":
			if cp.selectedIndex < len(cp.actions)-1 {
				cp.selectedIndex++
			}
			return cp, nil
		}
	}

	// Update filter input
	var cmd tea.Cmd
	cp.filterInput, cmd = cp.filterInput.Update(msg)

	// Re-filter actions based on input
	cp.filterActions()

	return cp, cmd
}

// View renders the command palette.
func (cp *CommandPalette) View() string {
	// Build header
	var header string
	if cp.sessionName != "" {
		header = theme.PaletteHeaderStyle.Render("Session: " + cp.sessionName)
	} else {
		header = theme.PaletteHeaderStyle.Render("Command Palette")
	}

	// Build action list
	var items []string
	maxNameLen := cp.maxActionNameLen()

	for i, action := range cp.actions {
		name := padRight(action.Name, maxNameLen)
		desc := action.Description

		if i == cp.selectedIndex {
			line := theme.PaletteItemSelectedStyle.Render("> "+name) +
				theme.PaletteDescSelectedStyle.Render("  "+desc)
			items = append(items, line)
		} else {
			line := theme.PaletteItemStyle.Render("  "+name) +
				theme.PaletteDescStyle.Render("  "+desc)
			items = append(items, line)
		}
	}

	// If no matches
	if len(items) == 0 {
		items = append(items, theme.PaletteDescStyle.Render("  No matching actions"))
	}

	actionList := strings.Join(items, "\n")

	// Build filter input
	filterLine := "> " + cp.filterInput.View()

	// Build footer
	footer := theme.PaletteFooterStyle.Render("↑↓ navigate • ⏎ select • esc cancel")

	// Combine all parts
	content := header + "\n" +
		strings.Repeat("─", cp.paletteWidth()-2) + "\n" +
		actionList + "\n" +
		strings.Repeat("─", cp.paletteWidth()-2) + "\n" +
		filterLine + "\n" +
		footer

	// Apply border
	bordered := theme.PaletteBorderStyle.Render(content)

	return bordered
}

// filterActions filters the action list based on the current input.
func (cp *CommandPalette) filterActions() {
	query := strings.ToLower(cp.filterInput.Value())
	if query == "" {
		cp.actions = cp.allActions
		cp.selectedIndex = 0
		return
	}

	var filtered []domain.Action
	for _, action := range cp.allActions {
		if fuzzyMatch(query, action.Name) || fuzzyMatch(query, action.Description) {
			filtered = append(filtered, action)
		}
	}

	cp.actions = filtered

	// Reset selection if out of bounds
	if cp.selectedIndex >= len(cp.actions) {
		cp.selectedIndex = 0
	}
}

// fuzzyMatch checks if all characters in query appear in order in target.
func fuzzyMatch(query, target string) bool {
	target = strings.ToLower(target)
	qi := 0
	for _, c := range target {
		if qi < len(query) && c == rune(query[qi]) {
			qi++
		}
	}
	return qi == len(query)
}

// maxActionNameLen returns the maximum action name length for alignment.
func (cp *CommandPalette) maxActionNameLen() int {
	maxLen := 0
	for _, action := range cp.actions {
		if len(action.Name) > maxLen {
			maxLen = len(action.Name)
		}
	}
	return maxLen
}

// paletteWidth returns the width of the palette content.
func (cp *CommandPalette) paletteWidth() int {
	// Fixed width or responsive based on terminal
	width := 60
	if cp.width > 0 && cp.width < 80 {
		width = cp.width - 10
	}
	if width < 40 {
		width = 40
	}
	return width
}

// padRight pads a string to the given width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
```

**Step 2: Verify compilation**

Run: `go build ./internal/ui/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/ui/command_palette.go
git commit -m "$(cat <<'EOF'
feat(ui): add command palette component

Bubble Tea component with:
- Fuzzy filtering on action name and description
- Keyboard navigation (up/down/enter/esc)
- Context-aware action filtering
- Header showing selected session name
EOF
)"
```

---

## Task 7: Integrate Command Palette into Model

**Files:**
- Modify: `internal/ui/model.go`

**Step 1: Add state constant**

In the `const` block for `uiState`, add (keeping alphabetical after stateCommentingSession):

```go
const (
	stateList uiState = iota
	stateCommandPalette  // ADD THIS
	stateCommentingSession
	stateConfirmingArchive
	// ... rest unchanged
)
```

**Step 2: Add field to Model struct**

In the `Model` struct (keep alphabetical):

```go
type Model struct {
	// ... existing fields
	commandPalette                         *CommandPalette          // Command palette overlay
	// ... rest of fields
}
```

**Step 3: Add case to Update switch**

In the `Update` method's switch statement:

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateList:
		return m.updateList(msg)
	case stateCommandPalette:
		return m.updateCommandPalette(msg)
	// ... rest unchanged
	}
}
```

**Step 4: Add updateCommandPalette method**

Add this new method:

```go
func (m *Model) updateCommandPalette(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Forward window size to palette
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
	}

	// Delegate to palette
	updated, cmd := m.commandPalette.Update(msg)
	m.commandPalette = updated.(*CommandPalette)

	// Check if palette completed
	if m.commandPalette.Completed {
		result := m.commandPalette.Result
		m.state = stateList
		m.commandPalette = nil

		if result.Cancelled || result.Action == nil {
			return m, m.sessionList.Init()
		}

		// Get selected session for dispatcher
		var session *ports.TmuxSession
		if item, ok := m.sessionList.list.SelectedItem().(SessionItem); ok {
			session = item.Session
		}

		// Dispatch the action
		dispatcher := NewActionDispatcher(session)
		actionMsg := dispatcher.Dispatch(*result.Action)

		if actionMsg != nil {
			// Process the action message through updateList
			return m.updateList(actionMsg)
		}

		return m, m.sessionList.Init()
	}

	return m, cmd
}
```

**Step 5: Handle ShowCommandPaletteMsg in updateList**

In `updateList`, add a case for `ShowCommandPaletteMsg`:

```go
case ShowCommandPaletteMsg:
	// Get selected session info for the palette header
	var session *ports.TmuxSession
	var sessionName string
	if item, ok := m.sessionList.list.SelectedItem().(SessionItem); ok {
		session = item.Session
		sessionName = item.DisplayName
	}

	m.commandPalette = NewCommandPalette(session, sessionName)
	m.state = stateCommandPalette

	// Send initial window size
	initCmd := m.commandPalette.Init()
	_, sizeCmd := m.commandPalette.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	return m, tea.Batch(initCmd, sizeCmd)
```

**Step 6: Handle new toggle messages in updateList**

Add cases for the new toggle messages:

```go
case ToggleTimestampsMsg:
	// Cycle timestamps (same logic as existing key handler)
	switch m.timestampMode {
	case TimestampRelative:
		m.timestampMode = TimestampAbsolute
	case TimestampAbsolute:
		m.timestampMode = TimestampHidden
	case TimestampHidden:
		m.timestampMode = TimestampRelative
	}
	m.sessionList.timestampMode = m.timestampMode
	refreshCmd := m.sessionList.RefreshFromState()
	return m, tea.Batch(refreshCmd, m.sessionList.Init())

case ToggleTokenChartMsg:
	m.tokenChart.Toggle()
	m.recalculateListHeight()
	return m, m.sessionList.Init()

case CycleStatusMsg:
	// Delegate to session list's cycleSessionStatus
	return m, m.sessionList.cycleSessionStatus(msg.SessionName)
```

**Step 7: Add View case for stateCommandPalette**

In the `View` method, add a case for the command palette with dimmed background:

```go
case stateCommandPalette:
	if m.commandPalette != nil {
		// Render dimmed background
		background := m.sessionList.View()
		if m.tokenChart.IsVisible() {
			background += "\n" + m.tokenChart.View() + "\n"
		}
		dimmed := applyDimOverlay(background)

		// Render palette centered
		palette := m.commandPalette.View()
		return compositeOverlay(dimmed, palette, m.width, m.height)
	}
```

**Step 8: Add helper functions for dimmed overlay**

Add these helper functions to model.go:

```go
// applyDimOverlay applies a dimmed style to all lines of the background.
func applyDimOverlay(background string) string {
	lines := strings.Split(background, "\n")
	for i, line := range lines {
		// Strip existing ANSI codes and apply dim style
		lines[i] = theme.DimmedStyle.Render(stripAnsi(line))
	}
	return strings.Join(lines, "\n")
}

// stripAnsi removes ANSI escape codes from a string.
func stripAnsi(s string) string {
	// Simple regex-free approach: skip escape sequences
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// compositeOverlay centers the palette over the dimmed background.
func compositeOverlay(background, palette string, width, height int) string {
	// Get palette dimensions
	paletteHeight := lipgloss.Height(palette)
	paletteWidth := lipgloss.Width(palette)

	// Calculate position (lower third, centered horizontally)
	topPadding := (height - paletteHeight) * 2 / 3
	if topPadding < 0 {
		topPadding = 0
	}
	leftPadding := (width - paletteWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	// Position the palette
	positioned := lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		palette,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(theme.ColorDimmed),
	)

	return positioned
}
```

**Step 9: Add strings import if not present**

Ensure `strings` is imported in model.go.

**Step 10: Verify compilation**

Run: `go build ./internal/ui/...`
Expected: No errors

**Step 11: Commit**

```bash
git add internal/ui/model.go
git commit -m "$(cat <<'EOF'
feat(ui): integrate command palette into model

Adds:
- stateCommandPalette UI state
- updateCommandPalette handler
- ShowCommandPaletteMsg handler
- Dimmed background overlay rendering
- Action dispatch via ActionDispatcher

The palette now appears with Shift+O (after key handler added).
EOF
)"
```

---

## Task 8: Add Key Handler in SessionList

**Files:**
- Modify: `internal/ui/session_list.go`

**Step 1: Add key handler**

In the `Update` method's key handling switch, add a case for the command palette key (add near other application-level keys):

```go
case key.Matches(msg, sl.keys.Application.CommandPalette.Binding):
	return sl, func() tea.Msg { return ShowCommandPaletteMsg{} }
```

**Step 2: Verify compilation**

Run: `go build ./internal/ui/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/ui/session_list.go
git commit -m "$(cat <<'EOF'
feat(ui): add Shift+O key handler for command palette

Completes the command palette integration by wiring up the
keyboard shortcut to emit ShowCommandPaletteMsg.
EOF
)"
```

---

## Task 9: Add Command Palette to Help Screen

**Files:**
- Modify: `internal/ui/help_screen.go`

**Step 1: Add command palette to help content**

In `buildHelpContent`, add the command palette shortcut to the Application section:

```go
// Application
content += "\n" + theme.HelpGroupStyle.Render("Application") + "\n"
content += renderBinding(keys.Application.CommandPalette.Binding)  // ADD THIS
content += renderBinding(keys.Application.Timestamps.Binding)
// ... rest unchanged
```

**Step 2: Verify compilation**

Run: `go build ./internal/ui/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/ui/help_screen.go
git commit -m "$(cat <<'EOF'
feat(ui): add command palette to help screen

Shows the Shift+O shortcut in the Application section of the help screen.
EOF
)"
```

---

## Task 10: Final Build and Manual Testing

**Step 1: Build with version info**

```bash
BRANCH=$(git branch --show-current)
go build -ldflags="-X main.Version=${BRANCH}-v1 \
  -X main.Commit=$(git rev-parse HEAD) \
  -X main.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X main.GoVersion=$(go version | awk '{print $3}')" \
  -o ./bin/rocha-${BRANCH}-v1 ./cmd
```

Expected: Binary created at `./bin/rocha-feat-command-palette-v3-v1`

**Step 2: Manual testing checklist**

Test the following manually:

1. **Open palette with Shift+O**
   - [ ] Palette appears with dimmed background
   - [ ] Header shows selected session name (if any)
   - [ ] Actions are listed alphabetically

2. **Filter actions**
   - [ ] Typing filters the action list
   - [ ] Fuzzy matching works (e.g., "ns" matches "new_session")
   - [ ] "No matching actions" shown when no matches

3. **Navigate actions**
   - [ ] Up/Down arrows move selection
   - [ ] Selection wraps at boundaries

4. **Execute action**
   - [ ] Enter executes the selected action
   - [ ] Session-scoped actions work correctly

5. **Close palette**
   - [ ] Esc closes without executing
   - [ ] Ctrl+C closes without executing

6. **Context-awareness**
   - [ ] When no session selected, session-scoped actions are hidden
   - [ ] When session selected, all actions are shown

7. **Help screen**
   - [ ] Shift+O shortcut appears in help

**Step 3: Commit any fixes**

If issues found, fix and commit appropriately.

---

## Summary

| Task | Files | Description |
|------|-------|-------------|
| 1 | `internal/domain/actions.go` | Domain action registry |
| 2 | `internal/ui/keys_*.go` | Key binding for Shift+O |
| 3 | `internal/ui/messages.go` | Message types |
| 4 | `internal/theme/*.go` | Styles for palette |
| 5 | `internal/ui/action_dispatcher.go` | Action-to-message mapping |
| 6 | `internal/ui/command_palette.go` | Palette component |
| 7 | `internal/ui/model.go` | Integration and overlay |
| 8 | `internal/ui/session_list.go` | Key handler |
| 9 | `internal/ui/help_screen.go` | Help screen update |
| 10 | - | Build and test |

Total: 10 tasks, ~45 steps
