package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
)

// PlaySoundCmd plays a notification sound
type PlaySoundCmd struct{}

// Run executes the sound playing logic
func (p *PlaySoundCmd) Run() error {
	return PlaySound()
}

// PlaySound plays a cross-platform notification sound (default)
func PlaySound() error {
	return PlaySoundForEvent("stop")
}

// PlaySoundForEvent plays different sounds based on the event type
func PlaySoundForEvent(eventType string) error {
	switch runtime.GOOS {
	case "darwin":
		return playMacOSForEvent(eventType)
	case "linux":
		return playLinuxForEvent(eventType)
	case "windows":
		return playWindowsForEvent(eventType)
	default:
		return terminalBell()
	}
}

func playMacOSForEvent(eventType string) error {
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
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return terminalBell()
}

func playLinuxForEvent(eventType string) error {
	var sounds []struct {
		cmd  string
		args []string
	}

	// Choose different sounds based on event type
	switch eventType {
	case "stop", "notification":
		// Claude finished or needs input
		sounds = []struct {
			cmd  string
			args []string
		}{
			{"paplay", []string{"/usr/share/sounds/freedesktop/stereo/complete.oga"}},
			{"aplay", []string{"/usr/share/sounds/freedesktop/stereo/complete.wav"}},
		}
	case "prompt":
		// User submitted prompt, Claude is working
		sounds = []struct {
			cmd  string
			args []string
		}{
			{"paplay", []string{"/usr/share/sounds/freedesktop/stereo/message.oga"}},
			{"aplay", []string{"/usr/share/sounds/freedesktop/stereo/message.wav"}},
			{"paplay", []string{"/usr/share/sounds/freedesktop/stereo/bell.oga"}},
		}
	case "start":
		// Session started
		sounds = []struct {
			cmd  string
			args []string
		}{
			{"paplay", []string{"/usr/share/sounds/freedesktop/stereo/service-login.oga"}},
			{"aplay", []string{"/usr/share/sounds/freedesktop/stereo/service-login.wav"}},
		}
	default:
		sounds = []struct {
			cmd  string
			args []string
		}{
			{"paplay", []string{"/usr/share/sounds/freedesktop/stereo/bell.oga"}},
			{"aplay", []string{"/usr/share/sounds/freedesktop/stereo/bell.wav"}},
		}
	}

	for _, sound := range sounds {
		cmd := exec.Command(sound.cmd, sound.args...)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return terminalBell()
}

func playWindowsForEvent(eventType string) error {
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

func terminalBell() error {
	// Terminal bell character
	fmt.Print("\a")
	return nil
}
