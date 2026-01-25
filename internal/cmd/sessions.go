package cmd

// SessionsCmd manages sessions
type SessionsCmd struct {
	Add       SessionsAddCmd       `cmd:"add" help:"Add a new session"`
	Archive   SessionsArchiveCmd   `cmd:"archive" help:"Archive or unarchive a session"`
	Comment   SessionsCommentCmd   `cmd:"comment" help:"Add, edit, or clear session comment"`
	Del       SessionsDelCmd       `cmd:"del" help:"Delete a session"`
	Duplicate SessionsDuplicateCmd `cmd:"duplicate" help:"Create session from existing repository"`
	Flag      SessionsFlagCmd      `cmd:"flag" help:"Toggle session flag"`
	List      SessionsListCmd      `cmd:"list" help:"List all sessions" default:"1"`
	Move      SessionsMoveCmd      `cmd:"move" aliases:"mv" help:"Move sessions between ROCHA_HOME directories"`
	Rename    SessionsRenameCmd    `cmd:"rename" help:"Update session display name"`
	Set       SessionSetCmd        `cmd:"set" help:"Set session configuration"`
	Status    SessionsStatusCmd    `cmd:"status" help:"Set or clear implementation status"`
	View      SessionsViewCmd      `cmd:"view" help:"View a specific session"`
}
