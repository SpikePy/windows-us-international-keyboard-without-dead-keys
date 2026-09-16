package setup

import "testing"

func TestCountdown(t *testing.T) {
	tests := []struct {
		elapsedMs uint32
		want      string
		due       bool
	}{
		{0, "Installing/updating automatically in 5 s...", false},
		{999, "Installing/updating automatically in 5 s...", false},
		{1000, "Installing/updating automatically in 4 s...", false},
		{4999, "Installing/updating automatically in 1 s...", false},
		{5000, "Installing/updating now...", true},
		{60000, "Installing/updating now...", true},
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
		{0, "Closing in 5 s...", false},
		{1500, "Closing in 4 s...", false},
		{4999, "Closing in 1 s...", false},
		{5000, "Closing now...", true},
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
