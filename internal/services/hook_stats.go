package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// ParseTimeString parses a time string into time.Time
// Supports:
// - RFC3339: "2026-01-26T10:00:00Z"
// - Relative: "1h ago", "30m ago", "2h30m ago"
// - Named: "today" (start of day), "now" (current time)
func ParseTimeString(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	s = strings.TrimSpace(s)

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Named shortcuts
	now := time.Now()
	switch strings.ToLower(s) {
	case "now":
		return now, nil
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}

	// Relative time (e.g., "1h ago", "30m ago", "2h30m ago")
	if strings.HasSuffix(s, " ago") {
		durationStr := strings.TrimSuffix(s, " ago")
		durationStr = strings.TrimSpace(durationStr)

		duration, err := parseRelativeDuration(durationStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid relative time format %q: %w", s, err)
		}

		return now.Add(-duration), nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format %q (use RFC3339, 'now', 'today', or '1h ago')", s)
}

// parseRelativeDuration parses duration strings like "1h", "30m", "2h30m"
func parseRelativeDuration(s string) (time.Duration, error) {
	// Try standard Go duration format first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Try parsing compound formats (e.g., "2h30m", "1h", "30m")
	// This is a fallback if the standard parser fails
	re := regexp.MustCompile(`(\d+)([hms])`)
	matches := re.FindAllStringSubmatch(s, -1)

	if len(matches) == 0 {
		return 0, fmt.Errorf("no valid duration components found")
	}

	var total time.Duration
	for _, match := range matches {
		value, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("invalid number %q", match[1])
		}

		switch match[2] {
		case "h":
			total += time.Duration(value) * time.Hour
		case "m":
			total += time.Duration(value) * time.Minute
		case "s":
			total += time.Duration(value) * time.Second
		}
	}

	return total, nil
}
