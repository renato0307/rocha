package sound

import "fmt"

// PlaySound plays a cross-platform notification sound (default)
func PlaySound() error {
	return PlaySoundForEvent("stop")
}

// PlaySoundForEvent plays different sounds based on the event type.
// Platform-specific implementations are in player_*.go files with build tags.
func PlaySoundForEvent(eventType string) error {
	return playForEvent(eventType)
}

// terminalBell outputs a terminal bell character as fallback
func terminalBell() error {
	fmt.Print("\a")
	return nil
}
