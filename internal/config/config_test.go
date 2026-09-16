package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useTempDir points Load and Path at a fresh directory and returns the
// config.yaml path they will use.
func useTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("LOCALAPPDATA", dir)
	path := filepath.Join(dir, "UndeadKeys", fileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// with returns the defaults with f applied.
func with(f func(*Config)) Config {
	c := defaults()
	f(&c)
	return c
}

func TestLoadCreatesDefaultFile(t *testing.T) {
	path := useTempDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg != defaults() {
		t.Errorf("first Load = %+v, want defaults %+v", cfg, defaults())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("default config.yaml was not created: %v", err)
	}
	for _, line := range []string{
		"\nstart_enabled: true\n",
		"\naltgr_shortcuts: true\n",
		"\nundead_keys: true\n",
		"\nrestrict_to_layout: \"00020409\"\n",
		"\nautostart: true\n",
	} {
		if !strings.Contains(string(data), line) {
			t.Errorf("created config.yaml is missing %q", strings.TrimSpace(line))
		}
	}

	again, err := Load()
	if err != nil || again != defaults() {
		t.Errorf("reloading the generated file = %+v, %v; want defaults, nil", again, err)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want Config
	}{
		{
			name: "file predating newer fields keeps their defaults",
			yaml: "start_enabled: false\n",
			want: with(func(c *Config) { c.StartEnabled = false }),
		},
		{
			name: "every field overridden",
			yaml: "start_enabled: false\naltgr_shortcuts: false\nundead_keys: false\nrestrict_to_layout: \"00000407\"\nautostart: false\n",
			want: Config{RestrictToLayout: "00000407"},
		},
		{
			name: "a quoted empty layout means every layout",
			yaml: "restrict_to_layout: \"\"\n",
			want: with(func(c *Config) { c.RestrictToLayout = "" }),
		},
		{
			name: "single quotes work too",
			yaml: "restrict_to_layout: ''\n",
			want: with(func(c *Config) { c.RestrictToLayout = "" }),
		},
		{
			name: "a layout key with no value keeps the default",
			yaml: "restrict_to_layout:\n",
			want: defaults(),
		},
		{
			name: "an unquoted layout ID",
			yaml: "restrict_to_layout: 0000041d\n",
			want: with(func(c *Config) { c.RestrictToLayout = "0000041d" }),
		},
		{
			name: "an invalid layout falls back to the default",
			yaml: "restrict_to_layout: \"US-International\"\nundead_keys: false\n",
			want: with(func(c *Config) { c.UndeadKeys = false }),
		},
		{
			name: "unknown keys are ignored",
			yaml: "start_enabled: false\nhotkey: ctrl+alt+u\n",
			want: with(func(c *Config) { c.StartEnabled = false }),
		},
		{
			name: "comments, blank lines and odd spacing",
			yaml: "# header\n\n  autostart:   no   # not on this PC\n\n# trailing note\nundead_keys: off\n",
			want: with(func(c *Config) { c.Autostart, c.UndeadKeys = false, false }),
		},
		{
			name: "windows line endings and a Notepad byte order mark",
			yaml: "\ufeff# edited in Notepad\r\nstart_enabled: false\r\nrestrict_to_layout: \"00000809\"\r\n",
			want: with(func(c *Config) { c.StartEnabled, c.RestrictToLayout = false, "00000809" }),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := useTempDir(t)
			if err := os.WriteFile(path, []byte(tt.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg != tt.want {
				t.Errorf("Load = %+v, want %+v", cfg, tt.want)
			}
		})
	}
}

func TestLoadRejectsBrokenFiles(t *testing.T) {
	tests := []struct{ name, yaml string }{
		{"value that isn't true or false", "autostart: maybe\n"},
		{"a list where a bool belongs", "start_enabled: [true\n"},
		{"a line that isn't key: value", "start_enabled true\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := useTempDir(t)
			if err := os.WriteFile(path, []byte(tt.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load()
			if err == nil {
				t.Error("Load returned no error")
			}
			if cfg != defaults() {
				t.Errorf("Load = %+v, want defaults %+v", cfg, defaults())
			}
		})
	}
}

func TestValidLayoutID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"00020409", true},
		{"0000041D", true},
		{"", true},
		{"0002040", false},
		{"000204090", false},
		{"0002040g", false},
		{"us-intl", false},
	}
	for _, tt := range tests {
		if got := ValidLayoutID(tt.in); got != tt.want {
			t.Errorf("ValidLayoutID(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestPath(t *testing.T) {
	want := useTempDir(t)
	got, err := Path()
	if err != nil || got != want {
		t.Errorf("Path() = %q, %v; want %q, nil", got, err, want)
	}
}
