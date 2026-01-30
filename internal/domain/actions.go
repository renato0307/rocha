package domain

// Action represents a user-invocable action in the system.
// This is the domain-level definition of what actions exist.
type Action struct {
	Description     string
	DisplayName     string // Human-readable name for UI display
	Name            string // Internal identifier for dispatching
	RequiresSession bool
}

// Actions is the canonical registry of all available actions.
// Sorted alphabetically by Name.
var Actions = []Action{
	{Name: "archive", DisplayName: "Archive", Description: "Hide session from the list", RequiresSession: true},
	{Name: "comment", DisplayName: "Comment", Description: "Add or edit session comment", RequiresSession: true},
	{Name: "cycle_status", DisplayName: "Cycle Status", Description: "Cycle through implementation statuses", RequiresSession: true},
	{Name: "flag", DisplayName: "Flag", Description: "Toggle session flag", RequiresSession: true},
	{Name: "help", DisplayName: "Help", Description: "Show keyboard shortcuts", RequiresSession: false},
	{Name: "kill", DisplayName: "Kill", Description: "Kill session and optionally remove worktree", RequiresSession: true},
	{Name: "new_from_repo", DisplayName: "New From Repo", Description: "Create session from same repository", RequiresSession: true},
	{Name: "new_session", DisplayName: "New Session", Description: "Create a new session", RequiresSession: false},
	{Name: "open", DisplayName: "Open", Description: "Attach to Claude session", RequiresSession: true},
	{Name: "open_editor", DisplayName: "Open Editor", Description: "Open session folder in editor", RequiresSession: true},
	{Name: "open_shell", DisplayName: "Open Shell", Description: "Open or attach to shell session", RequiresSession: true},
	{Name: "quit", DisplayName: "Quit", Description: "Exit Rocha", RequiresSession: false},
	{Name: "rename", DisplayName: "Rename", Description: "Rename session", RequiresSession: true},
	{Name: "send_text", DisplayName: "Send Text", Description: "Send text to session (experimental)", RequiresSession: true},
	{Name: "set_status", DisplayName: "Set Status", Description: "Choose implementation status", RequiresSession: true},
	{Name: "timestamps", DisplayName: "Timestamps", Description: "Toggle timestamp display", RequiresSession: false},
	{Name: "token_chart", DisplayName: "Token Chart", Description: "Toggle token usage chart", RequiresSession: false},
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
