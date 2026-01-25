//go:build darwin

package sound

import "os/exec"

// playForEvent plays sounds on macOS using afplay
func playForEvent(eventType string) error {
	var soundFiles []string

	// Choose different sounds based on event type
	switch eventType {
	case "stop", "notification":
		// Claude finished or needs input - calm, completion sound
		soundFiles = []string{
			"/System/Library/Sounds/Glass.aiff",
			"/System/Library/Sounds/Tink.aiff",
		}
	case "prompt":
		// User submitted prompt, Claude is working - active sound
		soundFiles = []string{
			"/System/Library/Sounds/Ping.aiff",
			"/System/Library/Sounds/Pop.aiff",
		}
	case "start":
		// Session started - welcoming sound
		soundFiles = []string{
			"/System/Library/Sounds/Submarine.aiff",
			"/System/Library/Sounds/Purr.aiff",
		}
	default:
		soundFiles = []string{"/System/Library/Sounds/Glass.aiff"}
	}

	// Try each sound file
	for _, soundFile := range soundFiles {
		cmd := exec.Command("afplay", soundFile)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}

	return terminalBell()
}
