package setup

import "fmt"

// Setup's dialog counts down twice: on its first page before running
// Install/Update on its own - so double-clicking Setup and walking away
// still installs or updates the tool - and on the result page of a
// successful action before closing itself. This file has no OS
// dependency, so its tests run anywhere.
const (
	AutoInstallSeconds = 5
	AutoCloseSeconds   = 5
)

// Countdown returns the line Setup's first page shows elapsedMs after it
// opened, and whether the wait is over and Install/Update should start.
func Countdown(elapsedMs uint32) (text string, due bool) {
	left := secondsLeft(AutoInstallSeconds, elapsedMs)
	if left <= 0 {
		return "Installing/updating now...", true
	}
	return fmt.Sprintf("Installing/updating automatically in %d s...", left), false
}

// CloseCountdown returns the line a successful result page shows
// elapsedMs after it appeared, and whether to close now.
func CloseCountdown(elapsedMs uint32) (text string, due bool) {
	left := secondsLeft(AutoCloseSeconds, elapsedMs)
	if left <= 0 {
		return "Closing now...", true
	}
	return fmt.Sprintf("Closing in %d s...", left), false
}

func secondsLeft(total int, elapsedMs uint32) int {
	return total - int(elapsedMs/1000)
}
