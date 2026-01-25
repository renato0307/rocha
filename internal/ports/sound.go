package ports

// SoundPlayer plays notification sounds
type SoundPlayer interface {
	// PlaySound plays the default notification sound
	PlaySound() error

	// PlaySoundForEvent plays a sound for a specific event type
	PlaySoundForEvent(eventType string) error
}
