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
| `-autostart` | from config.yaml | Whether the Startup shortcut should exist. The installed copy applies it every time it starts. |
| `-enable-logging` | off | Append diagnostics to `UndeadKeys.log` next to the exe. |

Booleans take the Go flag form: `-start-enabled=false` to turn one off.

### `Setup_UndeadKeys.exe`

| Flag | What it does |
|------|--------------|
| `-mode install\|uninstall` | Skip the dialog and run that action - for scripting. Progress goes to the console it was started from (or wherever its output is redirected), and the exit code is non-zero on failure. |
| `-install-dir <dir>` | Install into (or remove from) this directory instead of `%LOCALAPPDATA%\UndeadKeys`. |
| `-no-launch` | Install/update without starting it now. Install only. |
| `-no-autostart` | Leave the Startup shortcut as it is instead of applying the `autostart` setting. Install only. |
| `-keep-files` | Uninstall only: remove the Startup shortcut and stop the process, but leave the installed files. |

Opened without `-mode`, Setup shows its dialog with **Install/Update**
(install a fresh copy, or update an existing one in place),
**Uninstall** (remove the program, its shortcut and its settings) and
**Close** (change nothing). If nothing is chosen within 5 seconds, it runs
Install/Update on its own - so double-clicking Setup and walking away
still installs or updates UndeadKeys - and a line on the first page
counts down to that. A failed run stays open so the error can be read.

## Configuration

`%LOCALAPPDATA%\UndeadKeys\config.yaml` is created with defaults and a
comment per setting the first time the program runs, so there is always a
real file to edit. Keys it doesn't recognize are ignored, so a file
written by an older (or newer) version keeps working, and a setting with
an invalid value falls back to that setting's default rather than
stopping the program - an unusable `restrict_to_layout`, for instance,
reverts to `00020409`. The file only looks like YAML: the program reads
its flat `key: value` lines itself (comments, quotes, blank lines and
Notepad's line endings are all fine), so a few settings don't pull in a
YAML library. To intercept under every layout, write
`restrict_to_layout: ""`; a key with nothing after it keeps its default.

`autostart` decides whether the shortcut in your Startup folder exists.
Setup applies it when it installs, and the installed `UndeadKeys.exe`
applies it again every time it starts - so after changing it, the next
start (or a quit and restart from the tray) is enough, no re-install
needed. A copy started from anywhere other than the install directory
leaves the shortcut alone.

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
rounded keycap with an "Á" on it - is drawn at runtime with no image or
font files (`internal/keyicon`, `internal/tray`): the keycap from
rounded rectangles and the letter from three hand-drawn outlines, all
supersampled for smooth edges. The program declares itself DPI aware and
renders the icon at exactly the size the taskbar shows at the current
display scaling, so Windows never stretches it. It re-adds itself when
Explorer restarts (the `TaskbarCreated` broadcast), which is what
otherwise makes tray icons disappear for good.

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
  -ldflags "-H=windowsgui -s -w -X main.version=dev" -o Setup_UndeadKeys.exe ./cmd/undeadkeys-setup
```

`-H=windowsgui` is what keeps either program from opening a console
window; Setup's version is stamped the same way in the release build. The
setup program still prints for `-mode` runs: it borrows the
console of whatever started it (see `useParentConsole` in `cmd/undeadkeys-setup/main.go`).

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
(`setup.manifest`): it asks for version 6 of the common controls, which
has the task dialog Setup is built from, turns on per-monitor DPI
awareness, and declares that it never
needs elevation. Regenerate the `.syso` after editing it too.

Tests fail if you forget: one re-renders the glyph and checks that every
frame appears in each committed `.syso`, another that each manifest is
embedded in its current form. Another one fails if the character tables
above stop matching the code.

## Package layout

| Package | What's in it |
|---------|--------------|
| `cmd/undeadkeys` | The tray program: flags, single-instance guard, autostart sync, opt-in log, tray menu, message loop. |
| `cmd/undeadkeys-setup` | Setup: its task dialog (`dialog.go`), flags and the `-mode` console path (`main.go`). |
| `internal/hook` | The character tables and what to do with a key (`hook.go`, no OS dependency), and the Win32 hook that feeds it (`hook_windows.go`). |
| `internal/config` | `config.yaml`: defaults, parsing, per-field fallback. |
| `internal/shortcut` | The Startup shortcut (`IShellLink`), kept in line with the `autostart` setting. |
| `internal/tray` | Tray icon, popup menu, Explorer-restart recovery. |
| `internal/keyicon` | Renders the glyph at any size, shared by the tray, Setup's dialog and the file icon. No OS dependency. |
| `internal/win32` | Win32 declarations shared between packages: window classes, the message loop, icons from images, DPI awareness. |
| `internal/setup` | Install/uninstall and the WinINet download (`setup.go`), plus the release URLs (`release.go`) and the auto-install countdown (`countdown.go`), both without OS dependency. |
| `tools/genicon` | Writes the glyph to a multi-resolution `.ico` (16 to 256px). |

The OS-independent parts are the ones with table tests, which is why
`go test ./...` passes on Linux CI; the Windows-only code is covered by
vet and a cross-compile. The only dependency is
`golang.org/x/sys/windows`.

## Downloads and CI

Two workflows:

- `.github/workflows/ci.yml` runs on every push and pull request to
  `main`: tests, vet, build. It publishes nothing.
- `.github/workflows/build.yml` runs on a `v*` tag: the same checks, then
  it builds both exes with the tag stamped in as the version and attaches
  them to a GitHub Release.

`Setup_UndeadKeys.exe` never uses the GitHub API, whose unauthenticated
rate limit (60 requests an hour per address) makes installs fail on
shared networks. It downloads
`https://github.com/SpikePy/windows-us-international-keyboard-without-dead-keys/releases/latest/download/UndeadKeys.exe`,
which GitHub redirects to the newest release, and learns that release's
version by requesting `.../releases/latest` without following the
redirect and reading the tag from its `Location` header. Both go through
WinINet, Windows' own HTTP stack - so it uses the system proxy settings
and Windows' certificate store, and keeps megabytes of Go TLS code out of
the exe.

Setup's window is a Windows task dialog (`TaskDialogIndirect`), so it
needs no GUI toolkit and draws nothing itself but the tool's icon. The
first page asks the question and offers Install/Update, Uninstall and
Close (Close is `IDCANCEL`, so Escape and the title bar's X do the same),
with the dialog's timer counting down to the automatic Install/Update
(the countdown text is in `internal/setup/countdown.go`, tested).
Choosing navigates to a progress page - a marquee bar, the current step as
its text, Close disabled - and then to a result page saying what to do
next, or the error. The action runs on a worker goroutine that talks to
the dialog only through `SendMessage`, and the page showing is passed to
the dialog callback as its reference data, so no state is shared between
the two. `TASKDIALOGCONFIG` and `TASKDIALOG_BUTTON` are 1-byte packed, so
`dialog.go` writes them into byte buffers at fixed offsets and keeps every
string they point to alive.

## One program, one mode

The blueprint these tools follow has the program do its main action when
opened and run as a tray service only with `-background`. UndeadKeys has
no one-off action - intercepting keys in the background is all it does -
so it has a single mode: opening `UndeadKeys.exe` starts the tray
service, the Startup shortcut runs it without arguments, and opening it
while a copy is already running quietly does nothing (the named mutex
lets only one copy run).

## Upgrading from the PowerShell install

Versions before v1.2.0 were installed by `install-autostart.ps1`, which
copied `us-international-without-dead-keys.exe` straight into the Startup
folder. `Setup_UndeadKeys.exe` stops that process and deletes that copy,
on both install and uninstall, so the old and the new copy can't end up
fighting over the same keystrokes.
