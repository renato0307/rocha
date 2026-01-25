package editor

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/renato0307/rocha/internal/logging"
)

// Opener implements ports.EditorOpener
type Opener struct{}

// NewOpener creates a new editor opener
func NewOpener() *Opener {
	return &Opener{}
}

// Open opens the specified directory in an editor
// Priority: cliEditor → $ROCHA_EDITOR → $VISUAL → $EDITOR → platform defaults
func (o *Opener) Open(path string, cliEditor string) error {
	return openInEditorImpl(path, cliEditor)
}

// Package-level function for backwards compatibility

// OpenInEditor opens the specified directory in an editor
// Priority: cliEditor → $ROCHA_EDITOR → $VISUAL → $EDITOR → platform defaults
func OpenInEditor(path string, cliEditor string) error {
	return openInEditorImpl(path, cliEditor)
}

func openInEditorImpl(path string, cliEditor string) error {
	if path == "" {
		return fmt.Errorf("no path provided")
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}

	editor, args := findEditor(path, cliEditor)
	if editor == "" {
		return fmt.Errorf("no suitable editor found. Set --editor flag, $ROCHA_EDITOR, $VISUAL, or $EDITOR")
	}

	logging.Logger.Info("Opening editor", "editor", editor, "path", path)

	cmd := exec.Command(editor, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start editor: %w", err)
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			logging.Logger.Warn("Editor exited with error", "error", err, "editor", editor)
		}
	}()

	return nil
}

func findEditor(path string, cliEditor string) (string, []string) {
	// 1. CLI flag takes precedence
	if cliEditor != "" {
		return cliEditor, []string{path}
	}

	// 2. Check ROCHA_EDITOR
	if editor := os.Getenv("ROCHA_EDITOR"); editor != "" {
		return editor, []string{path}
	}

	// 3. Check VISUAL
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor, []string{path}
	}

	// 4. Check EDITOR
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, []string{path}
	}

	// 5. Platform-specific defaults
	return findPlatformEditor(path)
}
