package services

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/renato0307/rocha/internal/ports"
	portsmocks "github.com/renato0307/rocha/internal/ports/mocks"
)

func TestGetTodayHourlyUsage_CacheMiss(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	now := time.Now()
	usage := []ports.TokenUsage{
		{InputTokens: 100, OutputTokens: 50, Timestamp: now},
	}
	reader.EXPECT().GetTodayUsage().Return(usage, nil)

	service := NewTokenStatsService(reader)

	hourly, err := service.GetTodayHourlyUsage()

	require.NoError(t, err)
	require.Len(t, hourly, 1)
	assert.Equal(t, now.Hour(), hourly[0].Hour)
	assert.Equal(t, 100, hourly[0].InputTokens)
	assert.Equal(t, 50, hourly[0].OutputTokens)
}

func TestGetTodayHourlyUsage_CacheHit(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	now := time.Now()
	usage := []ports.TokenUsage{
		{InputTokens: 100, OutputTokens: 50, Timestamp: now},
	}
	// Only expect one call - second call should use cache
	reader.EXPECT().GetTodayUsage().Return(usage, nil)

	service := NewTokenStatsService(reader)

	// First call - cache miss
	hourly1, err := service.GetTodayHourlyUsage()
	require.NoError(t, err)

	// Second call - should hit cache
	hourly2, err := service.GetTodayHourlyUsage()
	require.NoError(t, err)

	assert.Equal(t, hourly1, hourly2)
}

func TestGetTodayHourlyUsage_ReaderError(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	reader.EXPECT().GetTodayUsage().Return(nil, errors.New("read error"))

	service := NewTokenStatsService(reader)

	hourly, err := service.GetTodayHourlyUsage()

	require.Error(t, err)
	assert.Nil(t, hourly)
}

func TestGetTodayHourlyUsage_HourlyAggregation(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	baseTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.Local)
	usage := []ports.TokenUsage{
		{InputTokens: 100, OutputTokens: 50, CacheCreation: 10, CacheRead: 5, Timestamp: baseTime.Add(9 * time.Hour)},           // 9:00
		{InputTokens: 200, OutputTokens: 100, CacheCreation: 20, CacheRead: 10, Timestamp: baseTime.Add(9*time.Hour + 30*time.Minute)}, // 9:30
		{InputTokens: 150, OutputTokens: 75, CacheCreation: 15, CacheRead: 8, Timestamp: baseTime.Add(10 * time.Hour)},          // 10:00
	}
	reader.EXPECT().GetTodayUsage().Return(usage, nil)

	service := NewTokenStatsService(reader)

	hourly, err := service.GetTodayHourlyUsage()

	require.NoError(t, err)
	require.Len(t, hourly, 2)

	// Hour 9 should have aggregated values
	assert.Equal(t, 9, hourly[0].Hour)
	assert.Equal(t, 300, hourly[0].InputTokens)  // 100 + 200
	assert.Equal(t, 150, hourly[0].OutputTokens) // 50 + 100
	assert.Equal(t, 30, hourly[0].CacheCreation) // 10 + 20
	assert.Equal(t, 15, hourly[0].CacheRead)     // 5 + 10

	// Hour 10
	assert.Equal(t, 10, hourly[1].Hour)
	assert.Equal(t, 150, hourly[1].InputTokens)
	assert.Equal(t, 75, hourly[1].OutputTokens)
}

func TestGetTodayHourlyUsage_SortedByHour(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	baseTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.Local)
	// Add out of order to verify sorting
	usage := []ports.TokenUsage{
		{InputTokens: 100, Timestamp: baseTime.Add(14 * time.Hour)}, // 14:00
		{InputTokens: 100, Timestamp: baseTime.Add(8 * time.Hour)},  // 8:00
		{InputTokens: 100, Timestamp: baseTime.Add(20 * time.Hour)}, // 20:00
	}
	reader.EXPECT().GetTodayUsage().Return(usage, nil)

	service := NewTokenStatsService(reader)

	hourly, err := service.GetTodayHourlyUsage()

	require.NoError(t, err)
	require.Len(t, hourly, 3)
	assert.Equal(t, 8, hourly[0].Hour)
	assert.Equal(t, 14, hourly[1].Hour)
	assert.Equal(t, 20, hourly[2].Hour)
}

func TestGetTodayTotals_CalculatesTotals(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	now := time.Now()
	usage := []ports.TokenUsage{
		{InputTokens: 100, OutputTokens: 50, CacheCreation: 10, CacheRead: 5, Timestamp: now},
		{InputTokens: 200, OutputTokens: 100, CacheCreation: 20, CacheRead: 10, Timestamp: now},
		{InputTokens: 150, OutputTokens: 75, CacheCreation: 15, CacheRead: 8, Timestamp: now},
	}
	reader.EXPECT().GetTodayUsage().Return(usage, nil)

	service := NewTokenStatsService(reader)

	totals, err := service.GetTodayTotals()

	require.NoError(t, err)
	assert.Equal(t, 450, totals.InputTokens)   // 100 + 200 + 150
	assert.Equal(t, 225, totals.OutputTokens)  // 50 + 100 + 75
	assert.Equal(t, 45, totals.CacheCreation)  // 10 + 20 + 15
	assert.Equal(t, 23, totals.CacheRead)      // 5 + 10 + 8
}

func TestGetTodayTotals_EmptyUsage(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	reader.EXPECT().GetTodayUsage().Return([]ports.TokenUsage{}, nil)

	service := NewTokenStatsService(reader)

	totals, err := service.GetTodayTotals()

	require.NoError(t, err)
	assert.Equal(t, 0, totals.InputTokens)
	assert.Equal(t, 0, totals.OutputTokens)
	assert.Equal(t, 0, totals.CacheCreation)
	assert.Equal(t, 0, totals.CacheRead)
}

func TestGetTodayTotals_ReaderError(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	reader.EXPECT().GetTodayUsage().Return(nil, errors.New("read error"))

	service := NewTokenStatsService(reader)

	totals, err := service.GetTodayTotals()

	require.Error(t, err)
	assert.Equal(t, ports.TokenTotals{}, totals)
}

func TestGetTodayTotals_SharesCacheWithHourly(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	now := time.Now()
	usage := []ports.TokenUsage{
		{InputTokens: 100, OutputTokens: 50, Timestamp: now},
	}
	// Only one call expected - both methods share cache
	reader.EXPECT().GetTodayUsage().Return(usage, nil)

	service := NewTokenStatsService(reader)

	// Call hourly first to populate cache
	_, err := service.GetTodayHourlyUsage()
	require.NoError(t, err)

	// Call totals - should use same cache
	totals, err := service.GetTodayTotals()

	require.NoError(t, err)
	assert.Equal(t, 100, totals.InputTokens)
	assert.Equal(t, 50, totals.OutputTokens)
}

func TestGetTodayHourlyUsage_EmptyUsage(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	reader.EXPECT().GetTodayUsage().Return([]ports.TokenUsage{}, nil)

	service := NewTokenStatsService(reader)

	hourly, err := service.GetTodayHourlyUsage()

	require.NoError(t, err)
	assert.Empty(t, hourly)
}

func TestGetTodayHourlyUsage_NilCache(t *testing.T) {
	reader := portsmocks.NewMockTokenUsageReader(t)

	reader.EXPECT().GetTodayUsage().Return(nil, nil)

	service := NewTokenStatsService(reader)

	hourly, err := service.GetTodayHourlyUsage()

	require.NoError(t, err)
	assert.Nil(t, hourly)
}
