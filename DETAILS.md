# UndeadKeys - details

Everything beyond [the README](README.md): the flags, the full character
tables, how the interception works, logging, and building from source.

## Flags

Every setting in `config.yaml` has a flag that overrides it for that run
only; the file itself is never rewritten.

### `UndeadKeys.exe`

| Flag | Default | What it does |
|------|---------|--------------|
| `-start-enabled` | from config.yaml | Intercept keys from the start, rather than starting paused. |
| `-altgr-shortcuts` | from config.yaml | Type accented characters on AltGr combinations. |
| `-undead-keys` | from config.yaml | Type the dead keys' plain characters immediately. |
| `-restrict-to-layout` | from config.yaml | Only intercept under this 8-hex-digit keyboard layout ID; `""` means every layout. |
| `-enable-logging` | off | Append diagnostics to `UndeadKeys.log` next to the exe. |

Booleans take the Go flag form: `-start-enabled=false` to turn one off.

### `Setup_UndeadKeys.exe`

| Flag | What it does |
|------|--------------|
| `-mode install\|uninstall` | Skip the window and run that action - for scripting. Progress goes to the console it was started from (or wherever its output is redirected), and the exit code is non-zero on failure. |
| `-install-dir <dir>` | Install into (or remove from) this directory instead of `%LOCALAPPDATA%\UndeadKeys`. |
| `-github-token <token>` | Use this token for the GitHub API lookup, to avoid the unauthenticated rate limit. Install only. |
| `-no-launch` | Install/update and register autostart, but don't start it now. Install only. |
| `-no-autostart` | Don't create or update the Startup shortcut. Install only. |
| `-keep-files` | Uninstall only: remove autostart and stop the process, but leave the installed files. |

## Configuration

`%LOCALAPPDATA%\UndeadKeys\config.yaml` is created with defaults and a
comment per setting the first time the program runs, so there is always a
real file to edit. Keys it doesn't recognize are ignored, so a file
written by an older (or newer) version keeps working, and a setting with
an invalid value falls back to that setting's default rather than
stopping the program - an unusable `restrict_to_layout`, for instance,
reverts to `00020409`.

Your keyboard layout's ID is the last eight hex digits shown by
`Get-WinUserLanguageList` in PowerShell, or under
`HKEY_CURRENT_USER\Keyboard Layout\Preload` in the registry.

## Character tables

While AltGr (Right Alt) is held:

| Key | AltGr | Shift+AltGr |
|-----|-------|-------------|
| `1` | ¡ | ¹ |
| `2` | ² | ² |
| `3` | ³ | ³ |
| `4` | ¤ | £ |
| `5` | € | € |
| `6` | ¼ | ¼ |
| `7` | ½ | ½ |
| `8` | ¾ | ¾ |
| `9` | ‘ | ‘ |
| `0` | ’ | ’ |
| `-` | ¥ | ¥ |
| `=` | × | ÷ |
| `Q` | ä | Ä |
| `W` | å | Å |
| `E` | é | É |
| `R` | ® | ® |
| `T` | þ | Þ |
| `Y` | ü | Ü |
| `U` | ú | Ú |
| `I` | í | Í |
| `O` | ó | Ó |
| `P` | ö | Ö |
| `[` | « | « |
| `]` | » | » |
| `\` | ¬ | ¦ |
| `A` | á | Á |
| `S` | ß | § |
| `D` | ð | Ð |
| `L` | ø | Ø |
| `;` | ¶ | ° |
| `'` | ´ | ¨ |
| `Z` | æ | Æ |
| `C` | © | ¢ |
| `N` | ñ | Ñ |
| `M` | µ | µ |
| `,` | ç | Ç |
| `/` | ¿ | ¿ |

These keys are dead keys under "United States-International" - normally
they swallow the keystroke and wait for a second one to combine into an
accent. They type their plain character immediately instead, whether or
not a modifier is held:

| Key | Alone | With Shift |
|-----|-------|------------|
| `'` | `'` | `"` |
| `` ` `` | `` ` `` | `~` |
| `6` | `6` | `^` |

A key in both tables (the apostrophe) follows the AltGr table while AltGr
is held, and the undead table otherwise.

## How it works

A low-level keyboard hook (`WH_KEYBOARD_LL`) sees every key before the
focused application does. For each key the program decides between three
outcomes (`internal/hook`): pass it through untouched, swallow it, or type
a character instead.

- **Layout check.** Unless `restrict_to_layout` is empty, a key is only
  ever intercepted while the focused window's layout ID matches.
  `GetKeyboardLayoutNameW` only reports the *calling* thread's layout, so
  the program briefly attaches to the foreground window's input thread to
  ask. That is far too expensive per keystroke, so the answer is cached
  per foreground window.
- **AltGr.** A low-level hook sees Right Alt as its own event, so the
  program tracks it across events rather than asking Windows whether it is
  held.
- **Typing the character.** The replacement goes out through `SendInput`
  as a `KEYEVENTF_UNICODE` event, which carries the character itself
  instead of a key to be translated - so the result doesn't depend on the
  active layout. Those synthetic events come back through the hook with
  virtual-key code 0, which no table maps, so they pass straight through.
- **Both halves.** For a key that is handled, the key-down types the
  character and the key-up is swallowed, so the application never sees
  half a keystroke of the raw key.

The tray icon is `Shell_NotifyIcon` on a hidden window. Its glyph - a
rounded keycap with an "Á" on it - is drawn at runtime with no image
files (`internal/keyicon`, `internal/tray`): the keycap from shapes,
supersampled for smooth edges, and the letter in Go Bold, a typeface
compiled into the program from `golang.org/x/image`. The program declares
itself DPI aware and renders the icon at exactly the size the taskbar
shows at the current display scaling, so Windows never stretches it. It re-adds
itself when Explorer restarts (the `TaskbarCreated` broadcast), which is
what otherwise makes tray icons disappear for good.

Everything is pure Go calling Win32 through `golang.org/x/sys/windows`:
no cgo, no GUI toolkit, so it cross-compiles to Windows from any OS and
ships as a single self-contained exe.

## Logging

Logging is off by default. With `-enable-logging`, `UndeadKeys.exe`
appends timestamped lines to `UndeadKeys.log` next to the exe - startup,
config problems, enable/disable from the tray, and any panic. A write
failure is ignored: logging must never be the reason the program
misbehaves.

## Building from source

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags "-H=windowsgui -s -w -X main.version=dev" \
  -o UndeadKeys.exe ./cmd/undeadkeys

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags "-H=windowsgui -s -w" -o Setup_UndeadKeys.exe ./cmd/undeadkeys-setup
```

`-H=windowsgui` is what keeps either program from opening a console
window. The setup program still prints for `-mode` runs: it borrows the
console of whatever started it (see `cmd/undeadkeys-setup/console.go`).

Tests, and the vet/build combination CI runs:

```sh
go test ./...
GOOS=windows GOARCH=amd64 go vet -unsafeptr=false ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...
```

`-unsafeptr=false` is needed because the `unsafe.Pointer` conversions of
addresses Win32 hands us (hook callback parameters, `CreateDIBSection`'s
output buffer) are outside anything the Go object graph tracks - inherent
to raw Win32 interop, and something vet's static heuristic can't tell
apart from a real misuse.

### The icon

Both exes carry the same glyph as their file icon, committed as a
`rsrc_windows_amd64.syso` per `cmd/` directory (`go build` picks those up
automatically). After changing `internal/keyicon`, regenerate them:

```sh
go run ./tools/genicon undeadkeys.ico
go run github.com/akavel/rsrc@latest -ico undeadkeys.ico -arch amd64 \
  -o cmd/undeadkeys/rsrc_windows_amd64.syso
go run github.com/akavel/rsrc@latest -ico undeadkeys.ico -arch amd64 \
  -manifest cmd/undeadkeys-setup/setup.manifest \
  -o cmd/undeadkeys-setup/rsrc_windows_amd64.syso
```

The setup program's `.syso` also carries its application manifest
(`setup.manifest`): it switches on the modern Windows control styles and
per-monitor DPI awareness for the setup window, and declares that it never
needs elevation. Regenerate the `.syso` after editing it too.

Tests fail if you forget: one re-renders the glyph and checks that every
frame appears in each committed `.syso`, another that each manifest is
embedded in its current form. Another one fails if the
character tables above stop matching the code.

## Package layout

| Package | What's in it |
|---------|--------------|
| `cmd/undeadkeys` | The tray program: flags, single-instance guard, tray menu, message loop. |
| `cmd/undeadkeys-setup` | The install/uninstall program: its window (plain Win32, no toolkit), and the `-mode` console path. |
| `internal/keymap` | The AltGr and undead character tables, and lookup. No OS dependency. |
| `internal/hook` | What to do with a key (`decide.go`, no OS dependency) and the Win32 hook that feeds it. |
| `internal/config` | `config.yaml`: defaults, loading, per-field fallback. |
| `internal/tray` | Tray icon, popup menu, runtime-drawn icon, Explorer-restart recovery. |
| `internal/keyicon` | Renders the glyph at any size, shared by the tray icon and the file icon. No OS dependency. |
| `internal/win32` | Win32 declarations shared between packages: window classes, the message loop, opening a file. |
| `internal/setup` | Install/uninstall, the Startup shortcut (`IShellLink`), and the WinINet download. |
| `internal/setupflow` | What the setup window shows and does: buttons, labels, the install countdown, closing itself. No OS dependency. |
| `internal/singleinstance` | The named-mutex guard. |
| `internal/applog` | The opt-in log file. |
| `tools/genicon` | Writes the glyph to a multi-resolution `.ico` (16 to 256px). |

The OS-independent packages are the ones with table tests, which is why
`go test ./...` passes on Linux CI; the Windows-only code is covered by
vet and a cross-compile.

## Downloads and CI

Two workflows:

- `.github/workflows/ci.yml` runs on every push and pull request to
  `main`: tests, vet, build. It publishes nothing.
- `.github/workflows/build.yml` runs on a `v*` tag: the same checks, then
  it builds both exes with the tag stamped in as the version and attaches
  them to a GitHub Release.

`Setup_UndeadKeys.exe` asks the GitHub API for the latest release and
downloads `UndeadKeys.exe` from it through WinINet, Windows' own HTTP
stack - so it uses the system proxy settings and Windows' certificate
store, and keeps megabytes of Go TLS code out of the exe.

## Upgrading from the PowerShell install

Versions before v1.2.0 were installed by `install-autostart.ps1`, which
copied `us-international-without-dead-keys.exe` straight into the Startup
folder. `Setup_UndeadKeys.exe` stops that process and deletes that copy,
on both install and uninstall, so the old and the new copy can't end up
fighting over the same keystrokes.
