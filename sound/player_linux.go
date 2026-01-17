//go:build linux

package sound

import "os/exec"

// playForEvent plays sounds on Linux using paplay (PulseAudio) or aplay (ALSA)
func playForEvent(eventType string) error {
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
