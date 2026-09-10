# winlayout-undead
inspired by umanovskis/win-kbd-usint-nodead and kdevo/winlayouts-undead

## Building and installing

Pushing changes to `US-AltGr-International.klc` (or running the workflow
manually from the Actions tab) triggers
`.github/workflows/build-keyboard-layout.yml`, which:

1. builds `tools/klc2c`, a small Go program that parses the `.klc` and
   generates the keyboard layout DLL's C source (`kbdusaltgr.c`/`.def`) -
   grounded in Microsoft's own `kbdus.c` reference sample and `kbd.h`'s real
   struct layout, rather than using MSKLC's `kbdutool.exe`, which currently
   crashes unreliably on GitHub's hosted Windows runner images;
2. compiles that generated source directly with MSVC (`cl.exe`/`link.exe`);
3. uploads an artifact named `US-AltGr-International-keyboard-layout`.

`kbd.h` is a minimal, self-authored header (not the WDK's) with just the
`KBDTABLES`-family definitions the generated source needs, so the whole
pipeline has no dependency on MSKLC or the WDK being downloaded anywhere.

The artifact contains:

- the compiled 64-bit and 32-bit layout DLLs
- `install-layout.ps1` / `uninstall-layout.ps1`
- a copy of the source `.klc`

To install on Windows 11: download and unzip the artifact, then run
`install-layout.ps1` from an elevated (Run as Administrator) PowerShell prompt.
Sign out and back in, then add the keyboard under Settings > Time & Language >
Language & region > English (United States) > Language options > Add a keyboard.
