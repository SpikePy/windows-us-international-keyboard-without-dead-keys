# windows-us-international-keyboard-without-dead-keys

A small background program for Windows that adds AltGr (Right Alt)
shortcuts for accented characters - no dead keys, no installation, no
Administrator rights needed.

Hold **Right Alt** and press a mapped key to get the accented character
immediately - nothing waits for a second keystroke.

**Only active while the Windows keyboard layout is
"United States-International"** - any other layout (German, plain US,
...) and this does nothing. Under that layout, `'`, `` ` ``/`~`, and
`6`/`^` are also always forced to their plain character immediately
instead of the dead-key wait that layout normally uses.

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

## Build

```
go build -buildvcs=false -ldflags="-H=windowsgui" -o us-international-without-dead-keys.exe .
```

Double-click the exe - no installer, no console window. Stop it via Task
Manager when done.

### Installation

Run `install-autostart.ps1` from an open PowerShell window (don't
double-click it - the window closes immediately on completion). It
downloads the latest release exe and copies it into your personal
Startup folder (`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`),
then starts it. No admin rights needed; re-running it just replaces the
existing copy, so it never leaves duplicates behind.

## How it works

A low-level keyboard hook (`WH_KEYBOARD_LL`) watches for Right Alt plus a
mapped key while the focused window's layout is
"United States-International" (checked via `GetKeyboardLayoutNameW`), and
injects the target Unicode character via `SendInput` instead. Everything
else passes through untouched. Pure Go (no cgo), so it cross-compiles to
Windows trivially from any OS.
