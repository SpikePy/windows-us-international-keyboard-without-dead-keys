// Package config reads the user-editable config.yaml that lives next to
// the installed app's data, in the same per-user directory the installer
// uses (%LOCALAPPDATA%\UndeadKeys). It is created with default values the
// first time it's loaded, so the user always has a real file to edit
// rather than having to know the option names up front.
//
// The file is YAML-shaped so editors highlight it and it reads the way
// people expect, but only the handful of "key: value" lines this tool
// writes are understood - see parse. A few flat settings don't justify a
// YAML library.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Defaults written to a freshly created config.yaml, and also the
// fallback used if the file is missing, unreadable, or has an invalid
// value for the corresponding field.
const (
	DefaultStartEnabled     = true
	DefaultAltGrShortcuts   = true
	DefaultUndeadKeys       = true
	DefaultRestrictToLayout = USInternationalKLID
	DefaultAutostart        = true
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
# Set it to "" to intercept under every layout instead. Can still be
# overridden per-run with -restrict-to-layout.
restrict_to_layout: "%s"

# autostart: whether UndeadKeys starts when you sign in to Windows (true),
# through a shortcut in your Startup folder, or not (false). Applied the
# next time the program starts, or when Setup installs it. Can still be
# overridden per-run with -autostart.
autostart: %t
`

// Config holds the settings read from config.yaml. Keys it doesn't know,
// including ones older or newer versions write, are ignored.
type Config struct {
	StartEnabled     bool
	AltGrShortcuts   bool
	UndeadKeys       bool
	RestrictToLayout string
	Autostart        bool
}

func defaults() Config {
	return Config{
		StartEnabled:     DefaultStartEnabled,
		AltGrShortcuts:   DefaultAltGrShortcuts,
		UndeadKeys:       DefaultUndeadKeys,
		RestrictToLayout: DefaultRestrictToLayout,
		Autostart:        DefaultAutostart,
	}
}

// Load reads config.yaml, creating it with default values on first run.
// A missing, unreadable or unparseable file falls back to the defaults,
// and an invalid value to that field's default, rather than failing - a
// bad config file should never stop the program from starting. A
// config.yaml written before a field existed is treated the same as that
// field being absent: the field keeps its default rather than being reset
// to zero.
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
			USInternationalKLID, DefaultRestrictToLayout, DefaultAutostart)
		if werr := os.WriteFile(path, []byte(text), 0o644); werr != nil {
			return def, fmt.Errorf("writing default config.yaml: %w", werr)
		}
		return def, nil
	}
	if err != nil {
		return def, fmt.Errorf("reading config.yaml: %w", err)
	}

	cfg := def
	if err := parse(data, &cfg); err != nil {
		return def, fmt.Errorf("parsing config.yaml: %w", err)
	}
	if !ValidLayoutID(cfg.RestrictToLayout) {
		cfg.RestrictToLayout = DefaultRestrictToLayout
	}
	return cfg, nil
}

// parse fills cfg from the file's "key: value" lines, leaving fields the
// file doesn't mention untouched. It understands blank lines, whole-line
// and trailing comments, optional quotes, and CRLF - everything a user
// editing this file in Notepad can produce. An unknown key is skipped so
// files from other versions still load; a true/false setting with any
// other value is an error.
func parse(data []byte, cfg *Config) error {
	text := strings.TrimPrefix(string(data), "\ufeff") // Notepad writes a BOM
	for n, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return fmt.Errorf(`line %d: expected "key: value", got %q`, n+1, line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if i := strings.Index(value, " #"); i >= 0 { // a trailing comment
			value = strings.TrimSpace(value[:i])
		}
		if value == "" { // "key:" with nothing after it: keep the default
			continue
		}
		// Only a quoted "" is an empty value; unquoting happens after
		// the check above so that stays distinguishable from no value.
		value = unquote(value)

		var err error
		switch key {
		case "start_enabled":
			cfg.StartEnabled, err = parseBool(value)
		case "altgr_shortcuts":
			cfg.AltGrShortcuts, err = parseBool(value)
		case "undead_keys":
			cfg.UndeadKeys, err = parseBool(value)
		case "restrict_to_layout":
			cfg.RestrictToLayout = value
		case "autostart":
			cfg.Autostart, err = parseBool(value)
		default:
			continue // a setting this version doesn't know
		}
		if err != nil {
			return fmt.Errorf("line %d: %s: %w", n+1, key, err)
		}
	}
	return nil
}

// unquote strips one pair of matching single or double quotes.
func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// parseBool accepts the spellings a hand-edited YAML-ish file may carry.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "yes", "on", "1":
		return true, nil
	case "false", "no", "off", "0":
		return false, nil
	}
	return false, fmt.Errorf("%q is not true or false", s)
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
