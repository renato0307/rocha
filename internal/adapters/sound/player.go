package sound

import "fmt"

// Player implements ports.SoundPlayer
type Player struct{}

// NewPlayer creates a new sound player
func NewPlayer() *Player {
	return &Player{}
}

// PlaySound plays a cross-platform notification sound (default)
func (p *Player) PlaySound() error {
	return p.PlaySoundForEvent("stop")
}

// PlaySoundForEvent plays different sounds based on the event type.
// Platform-specific implementations are in player_*.go files with build tags.
func (p *Player) PlaySoundForEvent(eventType string) error {
	return playForEvent(eventType)
}

// Package-level functions for backwards compatibility

// PlaySound plays a cross-platform notification sound (default)
func PlaySound() error {
	return PlaySoundForEvent("stop")
}

// PlaySoundForEvent plays different sounds based on the event type.
func PlaySoundForEvent(eventType string) error {
	return playForEvent(eventType)
}

// terminalBell outputs a terminal bell character as fallback
func terminalBell() error {
	fmt.Print("\a")
	return nil
}
