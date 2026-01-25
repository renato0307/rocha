package cmd

// PlaySoundCmd plays a notification sound
type PlaySoundCmd struct{}

// Run executes the sound playing logic
func (p *PlaySoundCmd) Run(cli *CLI) error {
	return cli.Container.NotificationService.PlaySound()
}
