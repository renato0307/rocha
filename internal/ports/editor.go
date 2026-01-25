package ports

// EditorOpener opens directories in an external editor
type EditorOpener interface {
	// Open opens the specified path in an editor
	// cliEditor is the editor specified via CLI flag (takes precedence)
	Open(path string, cliEditor string) error
}
