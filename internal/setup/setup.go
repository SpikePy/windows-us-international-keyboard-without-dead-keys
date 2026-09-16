//go:build windows

// Package setup implements what Setup_UndeadKeys.exe does: downloading
// UndeadKeys.exe, installing it under the user's own profile and keeping
// its autostart shortcut in line with the setting, and reversing all of
// that. The release links it downloads from are in release.go.
package setup

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"windows-us-international-keyboard-without-dead-keys/internal/config"
	"windows-us-international-keyboard-without-dead-keys/internal/shortcut"
)

const (
	// installDirName is the per-user directory under %LOCALAPPDATA%, the
	// same one config.yaml lives in.
	installDirName = "UndeadKeys"

	// legacyExeName is what versions before v1.2.0 ran, from the Startup
	// folder. Its process is stopped on install and uninstall so the old
	// and the new copy never fight over the same keystrokes (the file
	// itself is removed by shortcut.SyncAutostart).
	legacyExeName = "us-international-without-dead-keys.exe"

	userAgent = "undeadkeys-setup"
)

// resolveInstallDir returns dir, or %LOCALAPPDATA%\UndeadKeys if dir is
// empty.
func resolveInstallDir(dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", fmt.Errorf("%%LOCALAPPDATA%% is not set")
	}
	return filepath.Join(base, installDirName), nil
}

// latestVersion returns the tag of the newest release, from where GitHub
// redirects its "latest release" page.
func latestVersion() (string, error) {
	location, err := redirectTarget(latestPageURL())
	if err != nil {
		return "", fmt.Errorf("looking up the latest release: %w", err)
	}
	return tagFromLocation(location)
}

// InstallOptions configures Install.
type InstallOptions struct {
	InstallDir  string // defaults to %LOCALAPPDATA%\UndeadKeys if empty
	NoLaunch    bool   // install/update without starting it now
	NoAutostart bool   // leave the Startup shortcut as it is
	// Progress, if set, receives a short sentence as each step begins.
	Progress func(step string)
}

// Install downloads the latest released UndeadKeys.exe, installs it under
// the current user's %LOCALAPPDATA%, makes the Startup shortcut match
// config.yaml's autostart setting, and (re)starts it - terminating any
// already-running copy first so the file can be replaced and so at most
// one copy is ever running. Safe to re-run to update in place: it ends up
// with at most one Startup shortcut and exactly one running instance (the
// program also refuses to start a second copy via a named mutex, so this
// is belt and suspenders). Nothing here needs administrator rights: it
// only writes inside the user's profile. It returns the release tag it
// installed.
func Install(opts InstallOptions) (string, error) {
	progress := reporter(opts.Progress)
	dir, err := resolveInstallDir(opts.InstallDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating install dir: %w", err)
	}
	targetPath := filepath.Join(dir, assetName)

	progress("Looking up the latest release...")
	tag, err := latestVersion()
	if err != nil {
		return "", err
	}

	progress(fmt.Sprintf("Downloading %s...", tag))
	tmpPath := targetPath + ".download"
	if err := downloadFile(latestAssetURL(assetName), tmpPath); err != nil {
		return "", fmt.Errorf("downloading %s: %w", assetName, err)
	}

	progress("Stopping the running copy...")
	if err := stopRunning(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	progress(fmt.Sprintf("Installing to %s...", dir))
	if err := replaceFile(tmpPath, targetPath); err != nil {
		return "", fmt.Errorf("installing: %w", err)
	}

	if !opts.NoAutostart {
		// A config.yaml that can't be read still yields the defaults,
		// which have autostart on.
		cfg, _ := config.Load()
		if cfg.Autostart {
			progress("Adding it to Startup...")
		} else {
			progress("Leaving it out of Startup (autostart: false)...")
		}
		if err := shortcut.SyncAutostart(cfg.Autostart, targetPath); err != nil {
			return "", fmt.Errorf("updating autostart: %w", err)
		}
	}

	if !opts.NoLaunch {
		progress("Starting UndeadKeys...")
		if err := exec.Command(targetPath).Start(); err != nil {
			return "", fmt.Errorf("starting %s: %w", targetPath, err)
		}
	}
	return tag, nil
}

// UninstallOptions configures Uninstall.
type UninstallOptions struct {
	InstallDir string // defaults to %LOCALAPPDATA%\UndeadKeys if empty
	KeepFiles  bool   // remove autostart and stop the process, but leave the installed files in place
	// Progress, if set, receives a short sentence as each step begins.
	Progress func(step string)
}

// Uninstall reverses Install: removes the Startup shortcut (including the
// pre-v1.2.0 copy in the Startup folder), terminates any running copy,
// and - unless KeepFiles - deletes the installed files and config.yaml.
func Uninstall(opts UninstallOptions) error {
	progress := reporter(opts.Progress)
	dir, err := resolveInstallDir(opts.InstallDir)
	if err != nil {
		return err
	}

	progress("Removing it from Startup...")
	if err := shortcut.SyncAutostart(false, ""); err != nil {
		return fmt.Errorf("removing autostart: %w", err)
	}

	progress("Stopping the running copy...")
	if err := stopRunning(); err != nil {
		return err
	}

	if !opts.KeepFiles {
		progress(fmt.Sprintf("Removing %s...", dir))
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("removing %s: %w", dir, err)
		}
	}
	return nil
}

// reporter returns progress, or a function that ignores its steps if
// there is none.
func reporter(progress func(string)) func(string) {
	if progress == nil {
		return func(string) {}
	}
	return progress
}

// stopRunning terminates this tool under either the current or the old
// exe name.
func stopRunning() error {
	for _, exe := range []string{assetName, legacyExeName} {
		if err := terminateRunning(exe); err != nil {
			return fmt.Errorf("stopping %s: %w", exe, err)
		}
	}
	return nil
}

// downloadFile saves url's content to destPath, removing the file again
// if the download fails partway.
func downloadFile(url, destPath string) error {
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if err := httpGet(url, nil, out); err != nil {
		out.Close()
		os.Remove(destPath)
		return err
	}
	return out.Close()
}

// replaceFile moves tmpPath onto targetPath, retrying briefly: the target
// may still be momentarily locked right after terminateRunning killed the
// process that had it open/mapped.
func replaceFile(tmpPath, targetPath string) error {
	var err error
	for i := 0; i < 10; i++ {
		if err = os.Rename(tmpPath, targetPath); err == nil {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	os.Remove(tmpPath)
	return err
}

// terminateRunning finds every running process whose image file name
// matches exeName (case-insensitively) and terminates it, waiting briefly
// for each to actually exit.
func terminateRunning(exeName string) error {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}
	defer windows.CloseHandle(snap)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	var pids []uint32
	for err = windows.Process32First(snap, &entry); err == nil; err = windows.Process32Next(snap, &entry) {
		if strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), exeName) {
			pids = append(pids, entry.ProcessID)
		}
	}

	for _, pid := range pids {
		h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, pid)
		if err != nil {
			continue // already gone, or no permission - nothing more we can do
		}
		windows.TerminateProcess(h, 0)
		windows.WaitForSingleObject(h, 5000)
		windows.CloseHandle(h)
	}
	return nil
}

// Downloads go through WinINet, Windows' own HTTP stack, rather than Go's
// net/http: that keeps several MB of TLS code out of the exe, and it uses
// the user's system proxy settings and Windows' certificate store.
var (
	modWininet = windows.NewLazySystemDLL("wininet.dll")

	procInternetOpenW       = modWininet.NewProc("InternetOpenW")
	procInternetSetOptionW  = modWininet.NewProc("InternetSetOptionW")
	procInternetOpenUrlW    = modWininet.NewProc("InternetOpenUrlW")
	procHttpQueryInfoW      = modWininet.NewProc("HttpQueryInfoW")
	procInternetReadFile    = modWininet.NewProc("InternetReadFile")
	procInternetCloseHandle = modWininet.NewProc("InternetCloseHandle")
)

const (
	internetOpenTypePreconfig = 0 // use the system's proxy configuration

	internetFlagReload         = 0x80000000 // always ask the server, never the cache
	internetFlagNoCacheWrite   = 0x04000000
	internetFlagNoAutoRedirect = 0x00200000
	internetFlagNoCookies      = 0x00080000
	internetFlagNoUI           = 0x00000200

	internetOptionConnectTimeout = 2
	internetOptionReceiveTimeout = 6

	httpQueryStatusCode = 19
	httpQueryLocation   = 33
	httpQueryFlagNumber = 0x20000000

	// httpTimeoutMs bounds connecting and each individual read, so a
	// stalled connection fails instead of hanging, while a slow but
	// progressing download is never cut off.
	httpTimeoutMs uint32 = 30_000
)

// request is an open WinINet request, with the session it belongs to.
type request struct{ inet, req uintptr }

func (r request) close() {
	procInternetCloseHandle.Call(r.req)
	procInternetCloseHandle.Call(r.inet)
}

// openURL sends a GET for url with the given extra headers (each
// "Name: value") and WinINet flags, and returns the open request together
// with its HTTP status.
func openURL(url string, headers []string, flags uintptr) (request, uint32, error) {
	agent, err := windows.UTF16PtrFromString(userAgent)
	if err != nil {
		return request{}, 0, err
	}
	inet, _, e := procInternetOpenW.Call(uintptr(unsafe.Pointer(agent)), internetOpenTypePreconfig, 0, 0, 0)
	if inet == 0 {
		return request{}, 0, fmt.Errorf("InternetOpenW: %w", e)
	}

	timeout := httpTimeoutMs
	for _, opt := range []uintptr{internetOptionConnectTimeout, internetOptionReceiveTimeout} {
		procInternetSetOptionW.Call(inet, opt, uintptr(unsafe.Pointer(&timeout)), unsafe.Sizeof(timeout))
	}

	urlPtr, err := windows.UTF16PtrFromString(url)
	if err != nil {
		procInternetCloseHandle.Call(inet)
		return request{}, 0, err
	}
	var headersPtr *uint16
	var headersLen uintptr
	if len(headers) > 0 {
		if headersPtr, err = windows.UTF16PtrFromString(strings.Join(headers, "\r\n")); err != nil {
			procInternetCloseHandle.Call(inet)
			return request{}, 0, err
		}
		headersLen = 0xFFFFFFFF // (DWORD)-1: the headers string is NUL-terminated
	}
	req, _, e := procInternetOpenUrlW.Call(inet,
		uintptr(unsafe.Pointer(urlPtr)), uintptr(unsafe.Pointer(headersPtr)), headersLen,
		internetFlagReload|internetFlagNoCacheWrite|internetFlagNoCookies|internetFlagNoUI|flags, 0)
	if req == 0 {
		procInternetCloseHandle.Call(inet)
		return request{}, 0, fmt.Errorf("InternetOpenUrlW: %w", e)
	}
	r := request{inet, req}

	var status uint32
	size := uint32(unsafe.Sizeof(status))
	if ok, _, e := procHttpQueryInfoW.Call(req, httpQueryStatusCode|httpQueryFlagNumber,
		uintptr(unsafe.Pointer(&status)), uintptr(unsafe.Pointer(&size)), 0); ok == 0 {
		r.close()
		return request{}, 0, fmt.Errorf("HttpQueryInfoW: %w", e)
	}
	return r, status, nil
}

// httpGet fetches url, following redirects, and copies the response body
// to w. Any status other than 200 OK is an error, which includes the start
// of the response body.
func httpGet(url string, headers []string, w io.Writer) error {
	r, status, err := openURL(url, headers, 0)
	if err != nil {
		return err
	}
	defer r.close()

	body := responseReader(r.req)
	if status != 200 {
		return fmt.Errorf("HTTP %d%s", status, errorSnippet(body))
	}
	_, err = io.Copy(w, body)
	return err
}

// redirectTarget requests url without following its redirect and returns
// the address it redirects to.
func redirectTarget(url string) (string, error) {
	r, status, err := openURL(url, nil, internetFlagNoAutoRedirect)
	if err != nil {
		return "", err
	}
	defer r.close()

	if status < 300 || status > 399 {
		return "", fmt.Errorf("HTTP %d from %s where a redirect was expected", status, url)
	}
	buf := make([]uint16, 2048)
	size := uint32(len(buf) * 2) // HttpQueryInfoW counts bytes
	if ok, _, e := procHttpQueryInfoW.Call(r.req, httpQueryLocation,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0); ok == 0 {
		return "", fmt.Errorf("HttpQueryInfoW(Location): %w", e)
	}
	return windows.UTF16ToString(buf), nil
}

// errorSnippet returns the start of an error response's body as one short
// line for an error message ("" if the body is empty). GitHub's error
// pages are whole HTML documents.
func errorSnippet(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 4096))
	s := strings.Join(strings.Fields(string(b)), " ")
	if s == "" {
		return ""
	}
	const limit = 200
	if runes := []rune(s); len(runes) > limit {
		s = string(runes[:limit]) + "..."
	}
	return ": " + s
}

// responseReader reads a WinINet request handle's response body.
type responseReader uintptr

func (h responseReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	var n uint32
	if r, _, e := procInternetReadFile.Call(uintptr(h),
		uintptr(unsafe.Pointer(&p[0])), uintptr(len(p)), uintptr(unsafe.Pointer(&n))); r == 0 {
		return 0, fmt.Errorf("InternetReadFile: %w", e)
	}
	if n == 0 {
		return 0, io.EOF
	}
	return int(n), nil
}
