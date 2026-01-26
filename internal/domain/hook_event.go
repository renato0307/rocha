package domain

import "time"

// HookEvent represents a Claude Code hook event that fired during a session
type HookEvent struct {
	Command     string
	HookEvent   string
	HookName    string
	SessionName string
	Timestamp   time.Time
}
