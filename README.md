# windows-us-international-keyboard-without-dead-keys

A small background program for Windows that adds AltGr (Right Alt)
shortcuts for accented characters - no dead keys, no installation, no
Administrator rights needed.

Hold the physical **Right Alt** key and press a mapped letter, digit, or
symbol, and the accented character appears immediately - nothing waits for
a second keystroke.

**This only activates while the active Windows keyboard layout is
"United States-International"** - switch to German, plain US, or any
other layout and this program does nothing, leaving normal typing
completely untouched. Under that layout, the apostrophe (`'`),
backtick/grave (`` ` ``/`~`), and `6`/`^` keys are also always forced to
their plain character immediately instead of the dead-key wait that
layout would normally use.

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

Download `windows-us-international-keyboard-without-dead-keys.exe` from the
[latest release](../../releases/latest), or build it yourself:

```
go build -buildvcs=false -ldflags="-H=windowsgui" -o windows-us-international-keyboard-without-dead-keys.exe .
```

Then just double-click the exe - no installer, no console window. Stop it
via Task Manager when you're done.

### Start automatically at login

Download `install-autostart.ps1` (from this repo or the
[latest release](../../releases/latest)), open PowerShell (don't just
double-click the script - a `.ps1` window closes immediately on
completion, which can make it look like nothing happened), `cd` to the
folder you put it in, and run:

```
./install-autostart.ps1
```

This downloads the latest release exe straight from GitHub and installs
it into your personal Startup folder
(`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`), then starts
it right away. No admin rights needed - it only touches your own per-user
Startup folder. Re-running it later fetches whatever is newest and
replaces the copy already there, so it never leaves duplicates behind.

## How it works

A low-level keyboard hook (`WH_KEYBOARD_LL`) watches for the Right Alt key
held together with one of the mapped keys; when it sees that combination,
and the focused window's active keyboard layout is
"United States-International" (checked via `GetKeyboardLayoutNameW`), it
swallows the keystroke and injects the target Unicode character via
`SendInput` instead. Every other key press, and everything while any other
layout is active, passes through untouched. It's pure Go (no cgo), so it
cross-compiles to Windows trivially from any OS.
