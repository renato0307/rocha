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
