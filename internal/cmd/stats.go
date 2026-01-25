package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/renato0307/rocha/internal/ports"
	"github.com/renato0307/rocha/internal/ui"
)

// StatsCmd shows token usage statistics
type StatsCmd struct {
	Format string `help:"Output format (table or chart)" default:"table" enum:"table,chart"`
}

// Run executes the stats command
func (s *StatsCmd) Run(container *Container) error {
	// Get today's hourly usage
	hourly, err := container.TokenStatsService.GetTodayHourlyUsage()
	if err != nil {
		return fmt.Errorf("failed to get token usage: %w", err)
	}

	// Get today's totals
	totals, err := container.TokenStatsService.GetTodayTotals()
	if err != nil {
		return fmt.Errorf("failed to get token totals: %w", err)
	}

	// Display based on format
	switch s.Format {
	case "chart":
		s.renderChart(hourly, totals)
	default:
		s.renderTable(hourly, totals)
	}

	return nil
}

// renderTable displays token usage in table format
func (s *StatsCmd) renderTable(hourly []ports.HourlyTokenUsage, totals ports.TokenTotals) {
	today := time.Now().Format("2006-01-02")
	fmt.Printf("Token Usage - %s\n\n", today)

	if len(hourly) == 0 {
		fmt.Println("No token data yet.")
		return
	}

	// Header
	fmt.Println("Hour     Input       Output      Total")
	fmt.Println(strings.Repeat("─", 45))

	// Data rows
	for _, h := range hourly {
		total := h.InputTokens + h.OutputTokens
		fmt.Printf("%02d:00    %-11s %-11s %s\n",
			h.Hour,
			formatNumber(h.InputTokens),
			formatNumber(h.OutputTokens),
			formatNumber(total))
	}

	// Total row
	fmt.Println(strings.Repeat("─", 45))
	totalSum := totals.InputTokens + totals.OutputTokens
	fmt.Printf("Total    %-11s %-11s %s\n",
		formatNumber(totals.InputTokens),
		formatNumber(totals.OutputTokens),
		formatNumber(totalSum))
}

// renderChart displays token usage as a bar chart
func (s *StatsCmd) renderChart(hourly []ports.HourlyTokenUsage, totals ports.TokenTotals) {
	today := time.Now().Format("2006-01-02")
	fmt.Printf("Token Usage - %s\n\n", today)

	if len(hourly) == 0 {
		fmt.Println("No token data yet.")
		return
	}

	fmt.Println(ui.RenderTokenChart(hourly, totals))
}

// formatNumber formats a number with comma separators
func formatNumber(n int) string {
	if n == 0 {
		return "0"
	}

	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Add comma separators
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
