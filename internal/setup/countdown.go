package setup

import "fmt"

// AutoInstallSeconds is how long Setup's dialog waits for a choice before
// running Install/Update on its own - so double-clicking Setup and walking
// away still installs or updates the tool. This file has no OS dependency,
// so its tests run anywhere.
const AutoInstallSeconds = 5

// Countdown returns the line Setup's first page shows elapsedMs after it
// opened, and whether the wait is over and Install/Update should start.
func Countdown(elapsedMs uint32) (text string, due bool) {
	left := AutoInstallSeconds - int(elapsedMs/1000)
	if left <= 0 {
		return "Installing now...", true
	}
	return fmt.Sprintf("Install/Update starts on its own in %d s unless you choose.", left), false
}
