# winlayout-undead
inspired by umanovskis/win-kbd-usint-nodead and kdevo/winlayouts-undead

## Building and installing

Pushing changes to `US-AltGr-International.klc` (or running the workflow
manually from the Actions tab) triggers
`.github/workflows/build-keyboard-layout.yml`, which:

1. builds `tools/klc2c`, a small Go program that parses the `.klc` and
   generates the keyboard layout DLL's C source into `dll/kbdusaltgr.c`/`.def` -
   grounded in Microsoft's own `kbdus.c` reference sample and `kbd.h`'s real
   struct layout, rather than using MSKLC's `kbdutool.exe`, which currently
   crashes unreliably on GitHub's hosted Windows runner images;
2. cross-compiles that generated source into real Windows PE DLLs using
   mingw-w64 (`gcc`/`windres`) - runs entirely on a Linux runner, no Windows
   image or MSVC needed at all;
3. uploads an artifact named `US-AltGr-International-keyboard-layout`.

`dll/` holds the C build dependencies: `kbd.h` is a minimal, self-authored
header (not the WDK's) with just the `KBDTABLES`-family definitions the
generated source needs, and `kbdusaltgr.rc` is its version-info resource -
so the whole pipeline has no dependency on MSKLC, the WDK, or Windows itself
being available anywhere in CI. The generated `kbdusaltgr.c`/`.def` land in
`dll/` too (gitignored) so `#include "kbd.h"` resolves locally without any
extra include path.

The artifact contains:

- the compiled 64-bit and 32-bit layout DLLs
- `install-layout.ps1` / `uninstall-layout.ps1`
- a copy of the source `.klc`

To install on Windows 11: download and unzip the artifact, then run
`install-layout.ps1` from an elevated (Run as Administrator) PowerShell prompt.
Sign out and back in, then add the keyboard under Settings > Time & Language >
Language & region > English (United States) > Language options > Add a keyboard.

## No Administrator rights?

A real keyboard layout DLL has to live in `System32` and the registry, both
of which require admin. `tools/altgrhook` is a small, pure-Go background
program (no admin needed) implementing the same "AltGr for accented
characters, no dead keys" behavior on top of whatever layout is already
active: hold the physical Right Alt key and press a mapped letter/digit/
symbol and the character appears immediately. It uses a low-level keyboard
hook (`WH_KEYBOARD_LL`) to intercept just those key combinations and
`SendInput` to inject the Unicode character.

`.github/workflows/build-altgrhook.yml` cross-compiles it (plain `go build`,
`GOOS=windows`, no cgo) and uploads it as the `AltGr-background-helper`
artifact. Download `altgrhook.exe` and just run it - no installation, no
console window. Stop it via Task Manager when you're done, or add a
shortcut to it in your Startup folder (`Win+R` -> `shell:startup`, no admin
needed) to have it start automatically at login.
