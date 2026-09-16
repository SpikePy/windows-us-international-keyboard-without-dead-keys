# UndeadKeys

A small background program for Windows that adds AltGr (Right Alt)
shortcuts for accented characters - no dead keys, no installation, no
administrator rights needed.

Hold **Right Alt** and press a mapped key to get the accented character
immediately: `AltGr+E` types `é`, `Shift+AltGr+E` types `É`. The keys the
"United States-International" layout normally holds back as dead keys -
`'`, `` ` `` and `6` - type their plain character straight away instead.

**By default it only acts while the Windows keyboard layout is
"United States-International".** Under any other layout (German, plain
US, ...) it does nothing at all, so normal typing there is untouched.

## Install

Download `Setup_UndeadKeys.exe` from the
[latest release](https://github.com/SpikePy/windows-us-international-keyboard-without-dead-keys/releases/latest)
and run it. It installs `UndeadKeys.exe` into `%LOCALAPPDATA%\UndeadKeys`,
starts it, and adds a shortcut to your own Startup folder so it comes back
at login. Run it again any time to update to the latest release.

To remove it, run the same `Setup_UndeadKeys.exe` and choose **Uninstall**.

## The programs

| Program | What it is |
|---------|------------|
| `UndeadKeys.exe` | The background program itself. No window - it lives in the notification area. Left-click the tray icon to pause and resume it, right-click for Enable, Disable, Configure and Exit. |
| `Setup_UndeadKeys.exe` | Installs, updates and uninstalls the above. Double-click it for a menu; it installs on its own if nothing is chosen within 5 seconds. |

## Settings

Settings live in `%LOCALAPPDATA%\UndeadKeys\config.yaml`, created with
comments and these defaults the first time the program runs. The tray
menu's **Configure** opens it; changes take effect the next time the
program starts.

| Setting | Default | What it does |
|---------|---------|--------------|
| `start_enabled` | `true` | Intercept keys from the start, rather than waiting to be enabled from the tray. |
| `altgr_shortcuts` | `true` | Type accented characters on AltGr combinations. |
| `undead_keys` | `true` | Type `'`, `` ` `` and `6` immediately instead of waiting as dead keys. |
| `restrict_to_layout` | `"00020409"` | Only intercept under this keyboard layout ID (`00020409` is US-International). Empty means every layout. |

## Good to know

- **Nothing needs administrator rights and nothing is written to the
  registry**: the program, its settings and its autostart shortcut all
  live in your own user profile, and uninstalling removes them.
- While an **elevated** window has focus, Windows does not let a
  non-elevated program see its keystrokes - so under an admin console the
  accents fall back to whatever the layout does on its own.
- Only one copy ever runs, and the setup program stops the running one
  before replacing it.

Flags, the full character tables, how it works, logging and building from
source: see [DETAILS.md](DETAILS.md).
