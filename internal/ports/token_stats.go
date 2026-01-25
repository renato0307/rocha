package ports

import "time"

// TokenUsage represents token usage data from a single message
type TokenUsage struct {
	CacheCreation int
	CacheRead     int
	InputTokens   int
	OutputTokens  int
	Timestamp     time.Time
}

// HourlyTokenUsage represents aggregated token usage for a specific hour
type HourlyTokenUsage struct {
	CacheCreation int
	CacheRead     int
	Hour          int // 0-23
	InputTokens   int
	OutputTokens  int
}

// TokenTotals represents cumulative token totals
type TokenTotals struct {
	CacheCreation int
	CacheRead     int
	InputTokens   int
	OutputTokens  int
}

// TokenUsageReader reads token usage data from Claude session files
type TokenUsageReader interface {
	// GetTodayUsage returns all token usage entries for today
	GetTodayUsage() ([]TokenUsage, error)
}
