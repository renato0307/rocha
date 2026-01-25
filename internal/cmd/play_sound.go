package cmd

// PlaySoundCmd plays a notification sound
type PlaySoundCmd struct{}

// Run executes the sound playing logic
func (p *PlaySoundCmd) Run() error {
	// Create container to get NotificationService
	// Pass nil for tmuxClient since sound doesn't need tmux
	container, err := NewContainer(nil)
	if err != nil {
		return err
	}
	defer container.Close()

	return container.NotificationService.PlaySound()
}
