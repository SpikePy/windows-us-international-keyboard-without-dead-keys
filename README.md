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
