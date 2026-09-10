# winlayout-undead
inspired by umanovskis/win-kbd-usint-nodead and kdevo/winlayouts-undead

## Building and installing

Pushing changes to `US-AltGr-International.klc` (or running the workflow manually
from the Actions tab) triggers `.github/workflows/build-keyboard-layout.yml`, which
compiles the layout using Microsoft's MSKLC/kbdutool and uploads an artifact
named `US-AltGr-International-keyboard-layout` containing:

- the compiled 64-bit and 32-bit layout DLLs
- `install-layout.ps1` / `uninstall-layout.ps1`
- a copy of the source `.klc`

To install on Windows 11: download and unzip the artifact, then run
`install-layout.ps1` from an elevated (Run as Administrator) PowerShell prompt.
Sign out and back in, then add the keyboard under Settings > Time & Language >
Language & region > English (United States) > Language options > Add a keyboard.
