//go:build linux

package editor

import (
	"os"
	"os/exec"
)

var defaultEditors = []string{
	"code",
	"code-insiders",
	"cursor",
	"codium",
	"subl",
	"zed",
}

func findPlatformEditor(path string) (string, []string) {
	for _, editor := range defaultEditors {
		if _, err := exec.LookPath(editor); err == nil {
			return editor, []string{path}
		}
	}

	// Fallback: open shell in directory
	shell := "/bin/sh"
	if s := os.Getenv("SHELL"); s != "" {
		shell = s
	}
	return shell, []string{"-c", "cd " + path + " && exec $SHELL"}
}
