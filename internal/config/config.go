// Package config reads the user-editable config.yaml that lives next to
// the installed app's data, in the same per-user directory the installer
// uses (%LOCALAPPDATA%\UndeadKeys). It is created with default values the
// first time it's loaded, so the user always has a real file to edit
// rather than having to know the option names up front.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Defaults written to a freshly created config.yaml, and also the
// fallback used if the file is missing, unreadable, or has an invalid
// value for the corresponding field.
const (
	DefaultStartEnabled     = true
	DefaultAltGrShortcuts   = true
	DefaultUndeadKeys       = true
	DefaultRestrictToLayout = USInternationalKLID
)

// USInternationalKLID is the 8-hex-digit keyboard layout identifier of
// Windows' built-in "United States-International" layout, as reported by
// GetKeyboardLayoutNameW.
const USInternationalKLID = "00020409"

const fileName = "config.yaml"

const template = `# UndeadKeys configuration
#
# start_enabled: whether key interception is active as soon as the
# program starts (true), or starts paused - every key passing through
# untouched - until enabled from the tray icon (false). Can still be
# overridden per-run with -start-enabled.
start_enabled: %t

# altgr_shortcuts: whether holding AltGr (Right Alt) and pressing a
# mapped key types the accented character directly (see the key tables in
# DETAILS.md). Can still be overridden per-run with -altgr-shortcuts.
altgr_shortcuts: %t

# undead_keys: whether the apostrophe, backtick and 6 keys type their
# plain character immediately, instead of the dead-key wait the
# "United States-International" layout normally applies to them. Can
# still be overridden per-run with -undead-keys.
undead_keys: %t

# restrict_to_layout: only intercept keys while the focused window's
# keyboard layout has this 8-hex-digit layout ID - "%s" is
# "United States-International", the layout this tool is written for.
# Leave it empty to intercept under every layout instead. Can still be
# overridden per-run with -restrict-to-layout.
restrict_to_layout: "%s"
`

// Config holds the settings read from config.yaml. Keys it doesn't know,
// including ones older or newer versions write, are ignored.
type Config struct {
	StartEnabled     bool   `yaml:"start_enabled"`
	AltGrShortcuts   bool   `yaml:"altgr_shortcuts"`
	UndeadKeys       bool   `yaml:"undead_keys"`
	RestrictToLayout string `yaml:"restrict_to_layout"`
}

func defaults() Config {
	return Config{
		StartEnabled:     DefaultStartEnabled,
		AltGrShortcuts:   DefaultAltGrShortcuts,
		UndeadKeys:       DefaultUndeadKeys,
		RestrictToLayout: DefaultRestrictToLayout,
	}
}

// Load reads config.yaml, creating it with default values on first run.
// Any error, or an invalid value for a given field, falls back to that
// field's default rather than failing - a bad or missing config file
// should never stop the program from starting. A config.yaml written
// before a field existed is treated the same as that field being absent:
// the field keeps its default rather than being reset to zero.
func Load() (Config, error) {
	def := defaults()

	dir, err := userDir()
	if err != nil {
		return def, err
	}
	path := filepath.Join(dir, fileName)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		text := fmt.Sprintf(template,
			DefaultStartEnabled, DefaultAltGrShortcuts, DefaultUndeadKeys,
			USInternationalKLID, DefaultRestrictToLayout)
		if werr := os.WriteFile(path, []byte(text), 0o644); werr != nil {
			return def, fmt.Errorf("writing default config.yaml: %w", werr)
		}
		return def, nil
	}
	if err != nil {
		return def, fmt.Errorf("reading config.yaml: %w", err)
	}

	cfg := def
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return def, fmt.Errorf("parsing config.yaml: %w", err)
	}
	if !ValidLayoutID(cfg.RestrictToLayout) {
		cfg.RestrictToLayout = DefaultRestrictToLayout
	}
	return cfg, nil
}

// ValidLayoutID reports whether s is something GetKeyboardLayoutNameW
// could return - eight hexadecimal digits - or the empty string, which
// means "don't restrict to a layout at all".
func ValidLayoutID(s string) bool {
	if s == "" {
		return true
	}
	if len(s) != 8 {
		return false
	}
	return strings.IndexFunc(s, func(r rune) bool {
		return !strings.ContainsRune("0123456789abcdefABCDEF", r)
	}) < 0
}

// Path returns the config.yaml path, creating its containing directory
// if necessary. It does not create the file itself - see Load.
func Path() (string, error) {
	dir, err := userDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func userDir() (string, error) {
	dir := os.Getenv("LOCALAPPDATA")
	if dir == "" {
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolving user config directory: %w", err)
		}
	}
	dir = filepath.Join(dir, "UndeadKeys")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}
	return dir, nil
}
