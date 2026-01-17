package cmd

import (
	"rocha/sound"
)

// PlaySoundCmd plays a notification sound
type PlaySoundCmd struct{}

// Run executes the sound playing logic
func (p *PlaySoundCmd) Run() error {
	return sound.PlaySound()
}

// PlaySound plays a cross-platform notification sound (default)
// Re-exported for backward compatibility
func PlaySound() error {
	return sound.PlaySound()
}

// PlaySoundForEvent plays different sounds based on the event type
// Re-exported for backward compatibility
func PlaySoundForEvent(eventType string) error {
	return sound.PlaySoundForEvent(eventType)
}
