package cmd

// SessionsCmd manages sessions
type SessionsCmd struct {
	Add     SessionsAddCmd     `cmd:"add" help:"Add a new session"`
	Archive SessionsArchiveCmd `cmd:"archive" help:"Archive or unarchive a session"`
	Del     SessionsDelCmd     `cmd:"del" help:"Delete a session"`
	List    SessionsListCmd    `cmd:"list" help:"List all sessions" default:"1"`
	Move    SessionsMoveCmd    `cmd:"move" aliases:"mv" help:"Move sessions between ROCHA_HOME directories"`
	Set     SessionSetCmd      `cmd:"set" help:"Set session configuration"`
	View    SessionsViewCmd    `cmd:"view" help:"View a specific session"`
}
