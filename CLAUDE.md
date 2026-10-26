# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

UndeadKeys: a Windows tray program (pure Go, no cgo) that intercepts keys with a low-level keyboard hook to add AltGr accent shortcuts and make the US-International dead keys (`'`, `` ` ``, `6`) type immediately. Plus `Setup_UndeadKeys.exe`, a task-dialog installer. User-facing docs are README.md; DETAILS.md covers flags, character tables, internals and build — read it before non-trivial changes. The project follows the `windows-go-tool` skill's blueprint (use that skill for releases).

## After every working change

Don't ask - once the verify checks pass, run the `windows-go-tool` skill's Releasing steps end to end: commit, push, tag the next version (pick patch/minor/major per the skill's rules), watch the release build, then install it on this PC with `Setup_UndeadKeys.exe -mode install` and verify it is running.

## Commands

Development happens on Linux/WSL, cross-compiling to Windows. Mirror CI:

```sh
go test ./...                                              # OS-independent packages only
GOOS=windows GOARCH=amd64 go vet -unsafeptr=false ./...     # covers the Windows-only code
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...
go test ./internal/hook -run TestDocumentedTablesMatch      # single test
```

Release builds (as in `.github/workflows/build.yml`, triggered by pushing a `v*` tag):

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H=windowsgui -s -w -X main.version=dev" -o UndeadKeys.exe ./cmd/undeadkeys
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H=windowsgui -s -w -X main.version=dev" -o Setup_UndeadKeys.exe ./cmd/undeadkeys-setup
```

`-unsafeptr=false` is intentional: unsafe.Pointer conversions of Win32-supplied addresses are inherent to the interop.

## Architecture

- **OS-independent vs. Windows-only split.** Logic that can be tested lives in files without a build tag (`internal/hook/hook.go`, `internal/config`, `internal/keyicon`, `internal/setup/release.go`, `internal/setup/countdown.go`) and has table tests; Win32 code is `//go:build windows` (`hook_windows.go`, `setup.go`, `tray`, `shortcut`, `win32`) and is only vetted/compiled in CI. Keep new logic on the testable side where possible.
- **Hook decision** (`internal/hook`): per key event, pass through / swallow / type a character. Replacement characters go out via `SendInput` with `KEYEVENTF_UNICODE`; those synthetic events re-enter the hook with VK 0 and pass through. Key-down types, key-up is swallowed. Right Alt state is tracked across events. The foreground window's layout ID is looked up by attaching to its input thread and cached per foreground window.
- **Config** (`internal/config`): `%LOCALAPPDATA%\UndeadKeys\config.yaml`, YAML-shaped but parsed by hand (flat `key: value` lines, no YAML library). Unknown keys ignored; invalid values fall back per field to defaults. Every setting has a per-run flag override; the file is never rewritten. Adding a setting means updating config defaults/comments, the flag in `cmd/undeadkeys`, and the tables in README.md and DETAILS.md.
- **Autostart** (`internal/shortcut`): a Startup-folder shortcut, synced to the `autostart` setting by both Setup and by the installed exe on every start (only when run from the install dir).
- **Setup** (`cmd/undeadkeys-setup`, `internal/setup`): `TaskDialogIndirect` dialog; structs are 1-byte packed so `dialog.go` writes them into byte buffers at fixed offsets and must keep pointed-to strings alive. Work runs on a goroutine that talks to the dialog only via `SendMessage`. Downloads use WinINet and the plain `releases/latest/download/...` URL — never the GitHub API (rate limits); the version comes from the `Location` header of `releases/latest`. `-mode` runs are console-only, attaching to the parent console.
- **Icon** (`internal/keyicon`): drawn at runtime from outlines (no image/font files), shared by tray, Setup dialog and the `.exe` file icon. All private window message numbers live in `internal/win32` so they can't collide.
- No admin rights, no registry writes; single instance enforced by a named mutex. Only dependency: `golang.org/x/sys`.

## Tests that guard non-code files

- `internal/hook` `TestDocumentedTablesMatch` parses the character tables in **DETAILS.md** — change a mapping in `hook.go` and the doc together.
- `tools/genicon` tests check the committed `cmd/*/rsrc_windows_amd64.syso` contain the current glyph and manifest. After changing `internal/keyicon` or `cmd/undeadkeys-setup/setup.manifest`, regenerate:

```sh
go run ./tools/genicon undeadkeys.ico
go run github.com/akavel/rsrc@latest -ico undeadkeys.ico -arch amd64 -o cmd/undeadkeys/rsrc_windows_amd64.syso
go run github.com/akavel/rsrc@latest -ico undeadkeys.ico -arch amd64 -manifest cmd/undeadkeys-setup/setup.manifest -o cmd/undeadkeys-setup/rsrc_windows_amd64.syso
```

## Conventions

- Comments are explanatory prose (why, not what), with a package doc comment on each file's package; match that density.
- Docs use a plain hyphen ` - ` rather than em dashes.
