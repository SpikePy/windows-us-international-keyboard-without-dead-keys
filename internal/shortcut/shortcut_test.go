//go:build windows

package shortcut

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Windows-only, so Linux CI skips it; it works in a temp folder, never in
// the real Startup folder.
func TestSyncInFollowsTheSetting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Target.exe")
	if err := os.WriteFile(target, []byte("MZ"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, linkName)
	exists := func() bool {
		_, err := os.Stat(link)
		return err == nil
	}

	if err := syncIn(dir, false, target); err != nil {
		t.Fatalf("disabling with no shortcut yet: %v", err)
	}
	if exists() {
		t.Fatal("disabled, yet a shortcut exists")
	}

	if err := syncIn(dir, true, target); err != nil {
		t.Fatalf("enabling: %v", err)
	}
	got, err := shortcutTarget(link)
	if err != nil || !strings.EqualFold(got, target) {
		t.Fatalf("shortcut points at %q (%v), want %q", got, err, target)
	}

	if err := syncIn(dir, true, target); err != nil {
		t.Fatalf("enabling again: %v", err)
	}
	if links, _ := filepath.Glob(filepath.Join(dir, "*.lnk")); len(links) != 1 {
		t.Fatalf("enabling twice left %d shortcuts, want 1", len(links))
	}

	if err := syncIn(dir, false, target); err != nil {
		t.Fatalf("disabling: %v", err)
	}
	if exists() {
		t.Fatal("disabled, yet the shortcut is still there")
	}
}
