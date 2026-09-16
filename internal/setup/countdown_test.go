package setup

import "testing"

func TestCountdown(t *testing.T) {
	tests := []struct {
		elapsedMs uint32
		want      string
		due       bool
	}{
		{0, "Install/Update starts on its own in 5 s unless you choose.", false},
		{999, "Install/Update starts on its own in 5 s unless you choose.", false},
		{1000, "Install/Update starts on its own in 4 s unless you choose.", false},
		{4999, "Install/Update starts on its own in 1 s unless you choose.", false},
		{5000, "Installing now...", true},
		{60000, "Installing now...", true},
	}
	for _, tt := range tests {
		got, due := Countdown(tt.elapsedMs)
		if got != tt.want || due != tt.due {
			t.Errorf("Countdown(%d) = %q, %v; want %q, %v", tt.elapsedMs, got, due, tt.want, tt.due)
		}
	}
}

func TestCloseCountdown(t *testing.T) {
	tests := []struct {
		elapsedMs uint32
		want      string
		due       bool
	}{
		{0, "This window closes in 3 s.", false},
		{1500, "This window closes in 2 s.", false},
		{2999, "This window closes in 1 s.", false},
		{3000, "Closing...", true},
	}
	for _, tt := range tests {
		got, due := CloseCountdown(tt.elapsedMs)
		if got != tt.want || due != tt.due {
			t.Errorf("CloseCountdown(%d) = %q, %v; want %q, %v", tt.elapsedMs, got, due, tt.want, tt.due)
		}
	}
}

// The countdown text is swapped in place without resizing the dialog, so
// every second's line must be the same length as the first.
func TestCountdownLinesKeepTheirLength(t *testing.T) {
	for name, tc := range map[string]struct {
		f       func(uint32) (string, bool)
		seconds int
	}{
		"install": {Countdown, AutoInstallSeconds},
		"close":   {CloseCountdown, AutoCloseSeconds},
	} {
		first, _ := tc.f(0)
		for ms := uint32(0); ms < uint32(tc.seconds)*1000; ms += 1000 {
			if got, _ := tc.f(ms); len(got) != len(first) {
				t.Errorf("%s at %dms = %q is %d bytes, the first line %d", name, ms, got, len(got), len(first))
			}
		}
	}
}
