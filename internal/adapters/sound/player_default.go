//go:build !darwin && !linux && !windows

package sound

// playForEvent falls back to terminal bell on unsupported platforms
func playForEvent(eventType string) error {
	return terminalBell()
}
