# altgrhook

A small background program for Windows that adds AltGr (Right Alt)
shortcuts for accented characters - no dead keys, no installation, no
Administrator rights needed.

Hold the physical **Right Alt** key and press a mapped letter, digit, or
symbol, and the accented character appears immediately - nothing waits for
a second keystroke. It works on top of whatever keyboard layout you
already have active.

The apostrophe (`'`), backtick/grave (`` ` ``/`~`), and `6`/`^` keys are
also always forced to their plain character immediately, even without
AltGr - some built-in Windows layouts (like "United States-International")
treat those as dead keys by default, and this overrides that regardless of
which layout is active.

| Key | AltGr | Shift+AltGr | | Key | AltGr | Shift+AltGr |
|-----|-------|-------------|-|-----|-------|-------------|
| 1   | ¡     | ¹           | | Q   | ä     | Ä           |
| 2   | ²     | ²           | | W   | å     | Å           |
| 3   | ³     | ³           | | E   | é     | É           |
| 4   | ¤     | £           | | R   | ®     | ®           |
| 5   | €     | €           | | T   | þ     | Þ           |
| 6   | ¼     | ¼           | | Y   | ü     | Ü           |
| 7   | ½     | ½           | | U   | ú     | Ú           |
| 8   | ¾     | ¾           | | I   | í     | Í           |
| 9   | '     | '           | | O   | ó     | Ó           |
| 0   | '     | '           | | P   | ö     | Ö           |
| -   | ¥     | ¥           | | [   | «     | «           |
| =   | ×     | ÷           | | ]   | »     | »           |
| A   | á     | Á           | | Z   | æ     | Æ           |
| S   | ß     | §           | | C   | ©     | ¢           |
| D   | ð     | Ð           | | N   | ñ     | Ñ           |
| L   | ø     | Ø           | | M   | µ     | µ           |
| ;   | ¶     | °           | | ,   | ç     | Ç           |
| '   | ´     | ¨           | | /   | ¿     | ¿           |
| \\  | ¬     | ¦           |

## Get it

Download `altgrhook.exe` from the
[latest release](../../releases/latest), or build it yourself:

```
go build -buildvcs=false -ldflags="-H=windowsgui" -o altgrhook.exe .
```

Then just double-click `altgrhook.exe` - no installer, no console window.
Stop it via Task Manager when you're done.

### Start automatically at login

Put `altgrhook.exe` and `install-autostart.ps1` in the same folder, open
PowerShell (don't just double-click the script - a `.ps1` window closes
immediately on completion, which can make it look like nothing happened),
`cd` to that folder, and run:

```
./install-autostart.ps1
```

This copies `altgrhook.exe` into your personal Startup folder
(`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`) and starts it
right away. No admin rights needed - it only touches your own per-user
Startup folder. Re-running it after updating `altgrhook.exe` just replaces
the copy already there, so it never leaves duplicates behind.

## How it works

A low-level keyboard hook (`WH_KEYBOARD_LL`) watches for the Right Alt key
held together with one of the mapped keys; when it sees that combination it
swallows the keystroke and injects the target Unicode character via
`SendInput` instead. Every other key press passes through untouched. It's
pure Go (no cgo), so it cross-compiles to Windows trivially from any OS.
