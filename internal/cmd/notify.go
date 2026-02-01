package cmd

// NotifyCmd is the container for notification subcommands
type NotifyCmd struct {
	Handle   NotifyHandleCmd   `cmd:"handle" help:"Handle notification event from Claude hooks" default:"withargs"`
	ShowLogs NotifyShowLogsCmd `cmd:"show-logs" help:"Display hook execution logs"`
}
