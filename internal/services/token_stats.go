package services

import (
	"sync"
	"time"

	"github.com/renato0307/rocha/internal/logging"
	"github.com/renato0307/rocha/internal/ports"
)

const (
	// tokenStatsCacheTTL is the duration to cache token stats before refreshing
	tokenStatsCacheTTL = 60 * time.Second
)

// TokenStatsService provides token usage statistics with caching
type TokenStatsService struct {
	cache       *tokenStatsCache
	cacheMu     sync.RWMutex
	lastRefresh time.Time
	reader      ports.TokenUsageReader
}

// tokenStatsCache holds cached token statistics
type tokenStatsCache struct {
	hourly []ports.HourlyTokenUsage
	totals ports.TokenTotals
}

// NewTokenStatsService creates a new TokenStatsService
func NewTokenStatsService(reader ports.TokenUsageReader) *TokenStatsService {
	return &TokenStatsService{
		reader: reader,
	}
}

// GetTodayHourlyUsage returns token usage aggregated by hour for today (cached)
func (s *TokenStatsService) GetTodayHourlyUsage() ([]ports.HourlyTokenUsage, error) {
	if err := s.ensureCacheFresh(); err != nil {
		return nil, err
	}

	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	if s.cache == nil {
		return nil, nil
	}
	return s.cache.hourly, nil
}

// GetTodayTotals returns cumulative token totals for today (cached)
func (s *TokenStatsService) GetTodayTotals() (ports.TokenTotals, error) {
	if err := s.ensureCacheFresh(); err != nil {
		return ports.TokenTotals{}, err
	}

	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	if s.cache == nil {
		return ports.TokenTotals{}, nil
	}
	return s.cache.totals, nil
}

// ensureCacheFresh refreshes the cache if it's stale or empty
func (s *TokenStatsService) ensureCacheFresh() error {
	s.cacheMu.RLock()
	cacheValid := s.cache != nil && time.Since(s.lastRefresh) < tokenStatsCacheTTL
	s.cacheMu.RUnlock()

	if cacheValid {
		return nil
	}

	return s.refreshCache()
}

// refreshCache fetches fresh data and updates the cache
func (s *TokenStatsService) refreshCache() error {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// Double-check after acquiring write lock
	if s.cache != nil && time.Since(s.lastRefresh) < tokenStatsCacheTTL {
		return nil
	}

	logging.Logger.Debug("Refreshing token stats cache")

	usage, err := s.reader.GetTodayUsage()
	if err != nil {
		logging.Logger.Warn("Failed to get today's token usage", "error", err)
		return err
	}

	// Build hourly aggregation and totals in one pass
	hourlyMap := make(map[int]*ports.HourlyTokenUsage)
	var totals ports.TokenTotals

	for _, u := range usage {
		hour := u.Timestamp.Hour()
		if _, exists := hourlyMap[hour]; !exists {
			hourlyMap[hour] = &ports.HourlyTokenUsage{Hour: hour}
		}

		hourlyMap[hour].CacheCreation += u.CacheCreation
		hourlyMap[hour].CacheRead += u.CacheRead
		hourlyMap[hour].InputTokens += u.InputTokens
		hourlyMap[hour].OutputTokens += u.OutputTokens

		totals.CacheCreation += u.CacheCreation
		totals.CacheRead += u.CacheRead
		totals.InputTokens += u.InputTokens
		totals.OutputTokens += u.OutputTokens
	}

	// Convert map to sorted slice (by hour)
	var hourly []ports.HourlyTokenUsage
	for hour := 0; hour < 24; hour++ {
		if h, exists := hourlyMap[hour]; exists {
			hourly = append(hourly, *h)
		}
	}

	s.cache = &tokenStatsCache{
		hourly: hourly,
		totals: totals,
	}
	s.lastRefresh = time.Now()

	logging.Logger.Debug("Token stats cache refreshed",
		"hours_with_data", len(hourly),
		"input", totals.InputTokens,
		"output", totals.OutputTokens)

	return nil
}
