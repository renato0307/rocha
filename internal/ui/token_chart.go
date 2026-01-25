package ui

import (
	"fmt"
	"strings"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/charmbracelet/lipgloss"

	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/services"
	"github.com/renato0307/rocha/internal/theme"
)

const (
	tokenChartHeight   = 6   // Height of the chart area
	tokenChartWidth    = 120 // Fixed width for 24 hours with 2 bars each
	tokenChartBarWidth = 2   // Bar width
	tokenChartBarGap   = 0   // No gap between in/out bars, gap added between hours
)

// RenderTokenChart renders a token usage chart with the given data.
// This is used by both the TUI and CLI to ensure consistent formatting.
func RenderTokenChart(hourly []ports.HourlyTokenUsage, totals ports.TokenTotals) string {
	var sb strings.Builder

	// Build a map of hourly data for quick lookup
	hourlyMap := make(map[int]ports.HourlyTokenUsage)
	for _, h := range hourly {
		hourlyMap[h.Hour] = h
	}

	// Find max values for scaling and display
	var maxVal float64
	var maxInput, maxOutput int
	for _, h := range hourly {
		if h.InputTokens > maxInput {
			maxInput = h.InputTokens
		}
		if h.OutputTokens > maxOutput {
			maxOutput = h.OutputTokens
		}
		if float64(h.InputTokens) > maxVal {
			maxVal = float64(h.InputTokens)
		}
		if float64(h.OutputTokens) > maxVal {
			maxVal = float64(h.OutputTokens)
		}
	}

	if maxVal == 0 {
		maxVal = 1 // Avoid division by zero
	}

	// Legend with arrows, totals, and max values
	inputTotal := formatTokenCount(totals.InputTokens)
	inputMaxStr := formatTokenCount(maxInput)
	outputTotal := formatTokenCount(totals.OutputTokens)
	outputMaxStr := formatTokenCount(maxOutput)

	legend := theme.TokenChartLegendStyle.Render("Usage: ") +
		theme.TokenInputStyle.Render("↑") +
		theme.TokenChartLegendStyle.Render(" input: "+inputTotal+" (max: "+inputMaxStr+")  ") +
		theme.TokenOutputStyle.Render("↓") +
		theme.TokenChartLegendStyle.Render(" output: "+outputTotal+" (max: "+outputMaxStr+")")

	sb.WriteString(legend)
	sb.WriteString("\n\n")

	// Create bar chart
	axisStyle := lipgloss.NewStyle().Foreground(theme.ColorMuted)
	labelStyle := lipgloss.NewStyle().Foreground(theme.ColorSubtle)
	chart := barchart.New(tokenChartWidth, tokenChartHeight,
		barchart.WithStyles(axisStyle, labelStyle),
	)
	chart.SetBarWidth(tokenChartBarWidth)
	chart.SetBarGap(tokenChartBarGap)
	chart.SetMax(maxVal)

	// Create styles for input (green) and output (blue)
	inputStyle := lipgloss.NewStyle().Foreground(theme.ColorTokenInput)
	outputStyle := lipgloss.NewStyle().Foreground(theme.ColorTokenOutput)

	// Push bar data for all 24 hours (input + output side by side)
	for hour := 0; hour < 24; hour++ {
		h := hourlyMap[hour] // Will be zero struct if not present

		// Input bar with hour label
		chart.Push(barchart.BarData{
			Label: fmt.Sprintf("%02d", hour),
			Values: []barchart.BarValue{
				{Name: "in", Value: float64(h.InputTokens), Style: inputStyle},
			},
		})
		// Output bar (no label, pairs with input)
		chart.Push(barchart.BarData{
			Label: "",
			Values: []barchart.BarValue{
				{Name: "out", Value: float64(h.OutputTokens), Style: outputStyle},
			},
		})
	}

	chart.Draw()
	sb.WriteString(chart.View())

	return sb.String()
}

// TokenChart displays a grouped bar chart of token usage by hour
type TokenChart struct {
	hourlyUsage  []ports.HourlyTokenUsage
	statsService *services.TokenStatsService
	totals       ports.TokenTotals
	visible      bool
}

// NewTokenChart creates a new TokenChart component
func NewTokenChart(statsService *services.TokenStatsService) *TokenChart {
	return &TokenChart{
		statsService: statsService,
		visible:      false,
	}
}

// SetVisible sets the visibility of the chart
func (tc *TokenChart) SetVisible(visible bool) {
	tc.visible = visible
	if visible {
		tc.Refresh()
	}
}

// IsVisible returns whether the chart is visible
func (tc *TokenChart) IsVisible() bool {
	return tc.visible
}

// Toggle toggles the visibility of the chart
func (tc *TokenChart) Toggle() {
	tc.SetVisible(!tc.visible)
}

// Height returns the total height of the chart component (including spacing after)
func (tc *TokenChart) Height() int {
	if !tc.visible {
		return 0
	}
	return lipgloss.Height(tc.View()) + 1 // +1 for blank row after chart
}

// Refresh reloads data from the stats service
func (tc *TokenChart) Refresh() {
	if tc.statsService == nil {
		return
	}

	hourly, err := tc.statsService.GetTodayHourlyUsage()
	if err != nil {
		tc.hourlyUsage = nil
		tc.totals = ports.TokenTotals{}
		return
	}
	tc.hourlyUsage = hourly

	totals, err := tc.statsService.GetTodayTotals()
	if err != nil {
		tc.totals = ports.TokenTotals{}
		return
	}
	tc.totals = totals
}

// View renders the token chart
func (tc *TokenChart) View() string {
	if !tc.visible {
		return ""
	}
	return RenderTokenChart(tc.hourlyUsage, tc.totals)
}

// formatTokenCount formats a token count with K/M suffixes
func formatTokenCount(count int) string {
	if count >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(count)/1_000_000)
	}
	if count >= 1_000 {
		return fmt.Sprintf("%.0fK", float64(count)/1_000)
	}
	return fmt.Sprintf("%d", count)
}
