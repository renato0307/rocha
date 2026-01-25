//go:build windows

package editor

import (
	"os/exec"
)

var defaultEditors = []string{
	"code.cmd",
	"code-insiders.cmd",
	"cursor.cmd",
}

func findPlatformEditor(path string) (string, []string) {
	for _, editor := range defaultEditors {
		if _, err := exec.LookPath(editor); err == nil {
			return editor, []string{path}
		}
	}

	return "cmd.exe", []string{"/K", "cd", "/D", path}
}
