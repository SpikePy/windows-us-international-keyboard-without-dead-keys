// Package setupmenu is the behaviour behind Setup_UndeadKeys.exe's
// window: which buttons it offers, what they say, the countdown that
// installs on its own when nobody is there, and when the window closes
// itself. The window only draws what View returns and feeds events back
// in, so all of this is plain data with no OS dependency, tested on any
// platform.
package setupmenu

import (
	"errors"
	"fmt"
)

// CountdownSeconds is how long the window waits for the user before
// installing/updating on its own - so double-clicking Setup and walking
// away still gets the tool installed.
const CountdownSeconds = 5

// AutoCloseSeconds is how long the window stays open after an
// auto-chosen action succeeded: nobody was there to choose, so there is
// likely nobody to close it either. A manual run, or a failed one, stays
// open until closed so the result can be read.
const AutoCloseSeconds = 3

// Action is what Setup can do.
type Action int

const (
	NoAction Action = iota
	Install
	Uninstall
)

// Phase is where the window is in its short life.
type Phase int

const (
	Choosing Phase = iota
	Working
	Succeeded
	Failed
)

// Command is something the window has to do in response to an event.
type Command int

const (
	Nothing Command = iota
	StartInstall
	StartUninstall
	Close
)

// Model is the window's whole state.
type Model struct {
	Installed bool   // UndeadKeys.exe is present in the install directory
	Autostart bool   // config.yaml's autostart setting
	Latest    string // the newest release's tag, once looked up; "" before
	Own       string // Setup's own version, shown until Latest is known
	Phase     Phase  //
	Action    Action // the action running, or last run
	Auto      bool   // Action was started by the countdown, not the user
	Countdown int    // seconds before installing on its own; 0 = off
	CloseIn   int    // seconds before closing on its own; 0 = off
	Step      string // the step Working is on
	Version   string // release installed by the last successful Install
	Err       error  // why the last action failed
}

// New is the window as it first opens. own is Setup's own version ("dev"
// for a local build), and autostart the current autostart setting.
func New(installed, autostart bool, own string) Model {
	return Model{Installed: installed, Autostart: autostart, Own: own, Phase: Choosing, Countdown: CountdownSeconds}
}

// LatestKnown records the newest release's tag once the background lookup
// has found it.
func (m Model) LatestKnown(tag string) Model {
	m.Latest = tag
	return m
}

// version is the release the window talks about: the newest one if known,
// otherwise the one Setup itself came from, or "" for a local build.
func (m Model) version() string {
	switch {
	case m.Latest != "":
		return m.Latest
	case m.Own != "" && m.Own != "dev":
		return m.Own
	}
	return ""
}

// Tick advances the countdowns by one second.
func (m Model) Tick() (Model, Command) {
	switch {
	case m.Phase == Choosing && m.Countdown > 0:
		m.Countdown--
		if m.Countdown == 0 {
			return m.start(Install, true)
		}
	case m.Phase == Succeeded && m.CloseIn > 0:
		m.CloseIn--
		if m.CloseIn == 0 {
			return m, Close
		}
	}
	return m, Nothing
}

// UserActive stops both countdowns: someone is at the window, so they
// decide.
func (m Model) UserActive() Model {
	m.Countdown = 0
	m.CloseIn = 0
	return m
}

// Choose is the user picking an action with a button.
func (m Model) Choose(a Action) (Model, Command) {
	if m.Phase == Working {
		return m, Nothing
	}
	if a == Uninstall && !m.Installed {
		return m, Nothing
	}
	return m.start(a, false)
}

func (m Model) start(a Action, auto bool) (Model, Command) {
	m.Phase, m.Action, m.Auto = Working, a, auto
	m.Countdown, m.CloseIn = 0, 0
	m.Step, m.Err = "", nil
	if a == Uninstall {
		return m, StartUninstall
	}
	return m, StartInstall
}

// Progress records the step the running action has reached.
func (m Model) Progress(step string) Model {
	if m.Phase == Working {
		m.Step = step
	}
	return m
}

// Finished records the running action's outcome. version is the release
// an Install put in place, and installed whether UndeadKeys.exe is
// present now.
func (m Model) Finished(version string, err error, installed bool) Model {
	if m.Phase != Working {
		return m
	}
	m.Installed = installed
	m.Step = ""
	if err != nil {
		m.Phase, m.Err = Failed, err
		return m
	}
	m.Phase = Succeeded
	if m.Action == Install {
		m.Version = version
	}
	if m.Auto {
		m.CloseIn = AutoCloseSeconds
	}
	return m
}

// CanClose reports whether closing the window is allowed now: not while
// an action is halfway through replacing files.
func (m Model) CanClose() bool { return m.Phase != Working }

// View is what the window shows.
type View struct {
	Title         string // the tool's name and version
	Heading       string // the line under the title
	Status        string
	StatusIsError bool
	Busy          bool // show the progress bar

	Primary        string // the install/update button
	PrimaryEnabled bool

	UninstallVisible bool
	UninstallEnabled bool

	Close        string
	CloseEnabled bool

	// CloseIsDefault makes Enter close the window instead of installing
	// again, once there is nothing left to do.
	CloseIsDefault bool
}

// View derives what to show from m.
func (m Model) View() View {
	v := View{
		Title:            "UndeadKeys",
		Heading:          "Accented characters on AltGr, no dead keys",
		Primary:          "Install",
		PrimaryEnabled:   true,
		UninstallVisible: m.Installed,
		UninstallEnabled: true,
		Close:            "Close",
		CloseEnabled:     true,
	}
	if m.Installed {
		v.Primary = "Update"
	}
	version := m.version()
	if version != "" {
		v.Title += " " + version
	}

	switch m.Phase {
	case Choosing:
		target := "the latest release"
		if m.Latest != "" {
			target = m.Latest
		}
		switch {
		case m.Installed:
			v.Status = fmt.Sprintf("UndeadKeys is installed. Update it to %s, or remove it.", target)
		case m.Autostart:
			v.Status = fmt.Sprintf("Installs %s for your user account and starts it with Windows. No administrator rights needed.", installName(m.Latest))
		default:
			v.Status = fmt.Sprintf("Installs %s for your user account (autostart is off in config.yaml). No administrator rights needed.", installName(m.Latest))
		}
		if m.Countdown > 0 {
			v.Primary = fmt.Sprintf("%s (%d)", v.Primary, m.Countdown)
		}

	case Working:
		v.Busy = true
		v.PrimaryEnabled, v.UninstallEnabled, v.CloseEnabled = false, false, false
		v.Status = m.Step
		if v.Status == "" {
			v.Status = "Working..."
		}
		// Keep the button that was pressed visible while its action runs,
		// even though an uninstall is about to make it disappear.
		if m.Action == Uninstall {
			v.UninstallVisible = true
		}

	case Succeeded:
		v.CloseIsDefault = true
		if m.Action == Uninstall {
			v.Status = "UndeadKeys has been removed."
		} else if m.Version != "" {
			v.Status = fmt.Sprintf("UndeadKeys %s is installed and running.", m.Version)
			v.Title = "UndeadKeys " + m.Version
		} else {
			v.Status = "UndeadKeys is installed and running."
		}
		if m.CloseIn > 0 {
			v.Close = fmt.Sprintf("Close (%d)", m.CloseIn)
		}

	case Failed:
		v.StatusIsError = true
		verb := "Installing"
		if m.Action == Uninstall {
			verb = "Removing"
		}
		v.Status = fmt.Sprintf("%s failed: %v", verb, errOrUnknown(m.Err))
	}
	return v
}

// installName is what an install will put in place.
func installName(latest string) string {
	if latest == "" {
		return "UndeadKeys"
	}
	return "UndeadKeys " + latest
}

func errOrUnknown(err error) error {
	if err == nil {
		return errors.New("unknown error")
	}
	return err
}
