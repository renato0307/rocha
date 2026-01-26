package ports

// ProcessInspector provides methods to inspect running processes
type ProcessInspector interface {
	// GetClaudeSettings retrieves the --settings JSON from a running Claude process for a session
	GetClaudeSettings(sessionName string) (string, error)
}
