//go:build darwin

package editor

import (
	"os"
	"os/exec"
)

var defaultEditors = []string{
	"code",
	"code-insiders",
	"cursor",
}

func findPlatformEditor(path string) (string, []string) {
	for _, editor := range defaultEditors {
		if _, err := exec.LookPath(editor); err == nil {
			return editor, []string{path}
		}
	}

	apps := []string{
		"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
		"/Applications/Cursor.app/Contents/Resources/app/bin/cursor",
	}

	for _, app := range apps {
		if _, err := os.Stat(app); err == nil {
			return app, []string{path}
		}
	}

	shell := "/bin/sh"
	if s := os.Getenv("SHELL"); s != "" {
		shell = s
	}
	return shell, []string{"-c", "cd " + path + " && exec $SHELL"}
}
