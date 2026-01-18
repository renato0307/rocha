//go:build !linux && !darwin && !windows

package editor

import "os"

func findPlatformEditor(path string) (string, []string) {
	shell := "/bin/sh"
	if s := os.Getenv("SHELL"); s != "" {
		shell = s
	}
	return shell, []string{"-c", "cd " + path + " && exec $SHELL"}
}
