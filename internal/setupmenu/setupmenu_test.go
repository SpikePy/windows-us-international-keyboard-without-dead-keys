package setupmenu

import (
	"errors"
	"strings"
	"testing"
)

// ticks advances m by n seconds and returns the first command any tick
// produced.
func ticks(m Model, n int) (Model, Command) {
	first := Nothing
	for i := 0; i < n; i++ {
		var c Command
		m, c = m.Tick()
		if first == Nothing {
			first = c
		}
	}
	return m, first
}

func TestCountdownInstallsOnItsOwn(t *testing.T) {
	m := New(false, true, "dev")
	if got := m.View().Primary; got != "Install (5)" {
		t.Errorf("first primary label = %q, want %q", got, "Install (5)")
	}

	m, c := ticks(m, CountdownSeconds-1)
	if c != Nothing {
		t.Fatalf("command before the countdown ran out = %v", c)
	}
	if got := m.View().Primary; got != "Install (1)" {
		t.Errorf("primary label with one second left = %q", got)
	}

	m, c = m.Tick()
	if c != StartInstall || m.Phase != Working || !m.Auto {
		t.Fatalf("after the countdown: command %v, phase %v, auto %v; want StartInstall, Working, true", c, m.Phase, m.Auto)
	}
}

func TestUserActivityStopsTheCountdown(t *testing.T) {
	m := New(true, true, "dev").UserActive()
	m, c := ticks(m, 2*CountdownSeconds)
	if c != Nothing || m.Phase != Choosing {
		t.Fatalf("after user activity the countdown still fired: %v, phase %v", c, m.Phase)
	}
	if got := m.View().Primary; got != "Update" {
		t.Errorf("primary label = %q, want %q", got, "Update")
	}
}

func TestUnattendedSuccessClosesItself(t *testing.T) {
	m, _ := ticks(New(false, true, "dev"), CountdownSeconds)
	m = m.Progress("Downloading v1.2.3...")
	if v := m.View(); !v.Busy || v.Status != "Downloading v1.2.3..." || v.CloseEnabled || v.PrimaryEnabled {
		t.Errorf("while working: %+v", v)
	}
	if m.CanClose() {
		t.Error("the window may be closed halfway through an install")
	}

	m = m.Finished("v1.2.3", nil, true)
	v := m.View()
	if v.Status != "UndeadKeys v1.2.3 is installed and running." || v.Close != "Close (3)" || !v.CloseIsDefault {
		t.Errorf("after success: %+v", v)
	}
	if !v.UninstallVisible || v.Primary != "Update" {
		t.Errorf("after installing, the buttons should offer Update and Uninstall: %+v", v)
	}

	m, c := ticks(m, AutoCloseSeconds)
	if c != Close {
		t.Errorf("command after the close countdown = %v, want Close", c)
	}
}

func TestManualSuccessStaysOpen(t *testing.T) {
	m, c := New(false, true, "dev").UserActive().Choose(Install)
	if c != StartInstall || m.Auto {
		t.Fatalf("Choose(Install) = %v, auto %v", c, m.Auto)
	}
	m = m.Finished("v1.2.3", nil, true)
	if _, c := ticks(m, 10); c != Nothing {
		t.Errorf("a manual run closed itself (%v)", c)
	}
	if got := m.View().Close; got != "Close" {
		t.Errorf("close label = %q, want no countdown", got)
	}
}

func TestFailureStaysOpenWithTheError(t *testing.T) {
	m, _ := ticks(New(false, true, "dev"), CountdownSeconds)
	m = m.Finished("", errors.New("HTTP 404"), false)
	v := m.View()
	if !v.StatusIsError || !strings.Contains(v.Status, "HTTP 404") || !strings.HasPrefix(v.Status, "Installing failed") {
		t.Errorf("after failure: %+v", v)
	}
	if !v.PrimaryEnabled || !v.CloseEnabled || v.CloseIsDefault {
		t.Errorf("after failure the user must be able to retry or close: %+v", v)
	}
	if _, c := ticks(m, 10); c != Nothing {
		t.Errorf("a failed run closed itself (%v)", c)
	}
}

func TestUninstall(t *testing.T) {
	if _, c := New(false, true, "dev").Choose(Uninstall); c != Nothing {
		t.Errorf("Uninstall was offered with nothing installed (%v)", c)
	}
	if v := New(false, true, "dev").View(); v.UninstallVisible {
		t.Error("the Uninstall button shows with nothing installed")
	}

	m, c := New(true, true, "dev").Choose(Uninstall)
	if c != StartUninstall || m.Countdown != 0 {
		t.Fatalf("Choose(Uninstall) = %v, countdown %d", c, m.Countdown)
	}
	if v := m.View(); !v.UninstallVisible || v.UninstallEnabled {
		t.Errorf("while uninstalling the button should stay visible but disabled: %+v", v)
	}

	m = m.Finished("", nil, false)
	v := m.View()
	if v.Status != "UndeadKeys has been removed." || v.UninstallVisible || v.Primary != "Install" {
		t.Errorf("after uninstalling: %+v", v)
	}

	m, _ = New(true, true, "dev").Choose(Uninstall)
	if v := m.Finished("", errors.New("access denied"), true).View(); !strings.HasPrefix(v.Status, "Removing failed: access denied") {
		t.Errorf("uninstall failure status = %q", v.Status)
	}
}

func TestNothingStartsWhileWorking(t *testing.T) {
	m, _ := New(true, true, "dev").Choose(Install)
	if _, c := m.Choose(Uninstall); c != Nothing {
		t.Errorf("a second action started while one was running (%v)", c)
	}
	if got := m.Progress("x").Finished("v1", nil, true).Progress("late").View().Status; strings.Contains(got, "late") {
		t.Errorf("progress after finishing changed the status to %q", got)
	}
}

func TestTitleAndStatusCarryTheVersion(t *testing.T) {
	tests := []struct {
		name       string
		model      Model
		wantTitle  string
		wantStatus string
	}{
		{
			name:       "a local build before the lookup",
			model:      New(false, true, "dev"),
			wantTitle:  "UndeadKeys",
			wantStatus: "Installs UndeadKeys for your user account and starts it with Windows. No administrator rights needed.",
		},
		{
			name:      "a released Setup shows its own version until the lookup",
			model:     New(false, true, "v1.2.3"),
			wantTitle: "UndeadKeys v1.2.3",
		},
		{
			name:       "the newest release wins once known",
			model:      New(false, true, "v1.2.3").LatestKnown("v1.3.0"),
			wantTitle:  "UndeadKeys v1.3.0",
			wantStatus: "Installs UndeadKeys v1.3.0 for your user account and starts it with Windows. No administrator rights needed.",
		},
		{
			name:       "installed: offers the update",
			model:      New(true, true, "v1.2.3").LatestKnown("v1.3.0"),
			wantTitle:  "UndeadKeys v1.3.0",
			wantStatus: "UndeadKeys is installed. Update it to v1.3.0, or remove it.",
		},
		{
			name:       "installed, lookup still running",
			model:      New(true, true, "dev"),
			wantStatus: "UndeadKeys is installed. Update it to the latest release, or remove it.",
		},
		{
			name:       "autostart off is said up front",
			model:      New(false, false, "dev").LatestKnown("v1.3.0"),
			wantStatus: "Installs UndeadKeys v1.3.0 for your user account (autostart is off in config.yaml). No administrator rights needed.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.model.View()
			if tt.wantTitle != "" && v.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", v.Title, tt.wantTitle)
			}
			if tt.wantStatus != "" && v.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", v.Status, tt.wantStatus)
			}
		})
	}
}

func TestTitleShowsWhatWasInstalled(t *testing.T) {
	m, _ := New(false, true, "dev").Choose(Install)
	if got := m.Finished("v1.4.0", nil, true).View().Title; got != "UndeadKeys v1.4.0" {
		t.Errorf("title after installing = %q", got)
	}
}
