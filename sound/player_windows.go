//go:build windows

package sound

import "os/exec"

// playForEvent plays sounds on Windows using PowerShell
func playForEvent(eventType string) error {
	var soundCommands []string

	// Choose different sounds based on event type
	switch eventType {
	case "stop", "notification":
		// Claude finished or needs input
		soundCommands = []string{
			"[System.Media.SystemSounds]::Asterisk.Play()",
			"[System.Media.SystemSounds]::Beep.Play()",
		}
	case "prompt":
		// User submitted prompt, Claude is working
		soundCommands = []string{
			"[System.Media.SystemSounds]::Exclamation.Play()",
			"[System.Media.SystemSounds]::Beep.Play()",
		}
	case "start":
		// Session started
		soundCommands = []string{
			"[System.Media.SystemSounds]::Question.Play()",
			"[System.Media.SystemSounds]::Beep.Play()",
		}
	default:
		soundCommands = []string{"[System.Media.SystemSounds]::Beep.Play()"}
	}

	for _, soundCmd := range soundCommands {
		cmd := exec.Command("powershell", "-c", soundCmd)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return terminalBell()
}
