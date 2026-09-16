package setupflow

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
	m := New(false)
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
	m := New(true).UserActive()
	m, c := ticks(m, 2*CountdownSeconds)
	if c != Nothing || m.Phase != Choosing {
		t.Fatalf("after user activity the countdown still fired: %v, phase %v", c, m.Phase)
	}
	if got := m.View().Primary; got != "Update" {
		t.Errorf("primary label = %q, want %q", got, "Update")
	}
}

func TestUnattendedSuccessClosesItself(t *testing.T) {
	m, _ := ticks(New(false), CountdownSeconds)
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
	m, c := New(false).UserActive().Choose(Install)
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
	m, _ := ticks(New(false), CountdownSeconds)
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
	if _, c := New(false).Choose(Uninstall); c != Nothing {
		t.Errorf("Uninstall was offered with nothing installed (%v)", c)
	}
	if v := New(false).View(); v.UninstallVisible {
		t.Error("the Uninstall button shows with nothing installed")
	}

	m, c := New(true).Choose(Uninstall)
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

	m, _ = New(true).Choose(Uninstall)
	if v := m.Finished("", errors.New("access denied"), true).View(); !strings.HasPrefix(v.Status, "Removing failed: access denied") {
		t.Errorf("uninstall failure status = %q", v.Status)
	}
}

func TestNothingStartsWhileWorking(t *testing.T) {
	m, _ := New(true).Choose(Install)
	if _, c := m.Choose(Uninstall); c != Nothing {
		t.Errorf("a second action started while one was running (%v)", c)
	}
	if got := m.Progress("x").Finished("v1", nil, true).Progress("late").View().Status; strings.Contains(got, "late") {
		t.Errorf("progress after finishing changed the status to %q", got)
	}
}
