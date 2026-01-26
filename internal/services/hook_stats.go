package services

import (
	"github.com/renato0307/rocha/internal/domain"
	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
)

// HookStatsService provides hook event statistics and filtering
type HookStatsService struct {
	reader ports.HookReader
}

// NewHookStatsService creates a new HookStatsService
func NewHookStatsService(reader ports.HookReader) *HookStatsService {
	return &HookStatsService{
		reader: reader,
	}
}

// GetHookEvents returns hook events matching the filter
func (s *HookStatsService) GetHookEvents(filter ports.HookFilter) ([]domain.HookEvent, error) {
	logging.Logger.Debug("Getting hook events",
		"session", filter.SessionName,
		"event_type", filter.EventType,
		"from", filter.From,
		"to", filter.To,
		"limit", filter.Limit)

	events, err := s.reader.GetHookEvents(filter)
	if err != nil {
		logging.Logger.Warn("Failed to get hook events", "error", err)
		return nil, err
	}

	logging.Logger.Debug("Retrieved hook events", "count", len(events))
	return events, nil
}
