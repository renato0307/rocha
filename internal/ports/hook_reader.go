package ports

import (
	"time"

	"github.com/renato0307/rocha/internal/domain"
)

// HookFilter specifies criteria for filtering hook events
type HookFilter struct {
	EventType   string
	From        time.Time
	Limit       int
	SessionName string
	To          time.Time
}

// HookReader reads hook events from Claude session files
type HookReader interface {
	// GetHookEvents returns hook events matching the filter
	GetHookEvents(filter HookFilter) ([]domain.HookEvent, error)
}
