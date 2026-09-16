package setup

import "fmt"

// Setup's dialog counts down twice when nobody is there: before running
// Install/Update on its own - so double-clicking Setup and walking away
// still installs or updates the tool - and, after such an unattended
// install succeeded, before closing itself. This file has no OS
// dependency, so its tests run anywhere.
const (
	AutoInstallSeconds = 5
	AutoCloseSeconds   = 3
)

// Countdown returns the line Setup's first page shows elapsedMs after it
// opened, and whether the wait is over and Install/Update should start.
func Countdown(elapsedMs uint32) (text string, due bool) {
	left := secondsLeft(AutoInstallSeconds, elapsedMs)
	if left <= 0 {
		return "Installing now...", true
	}
	return fmt.Sprintf("Install/Update starts on its own in %d s unless you choose.", left), false
}

// CloseCountdown returns the line the result page of an unattended
// install shows elapsedMs after it appeared, and whether to close now.
func CloseCountdown(elapsedMs uint32) (text string, due bool) {
	left := secondsLeft(AutoCloseSeconds, elapsedMs)
	if left <= 0 {
		return "Closing...", true
	}
	return fmt.Sprintf("This window closes in %d s.", left), false
}

func secondsLeft(total int, elapsedMs uint32) int {
	return total - int(elapsedMs/1000)
}
