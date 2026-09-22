// Package usb writes a hyprtk ISO to a USB stick and prepares the optional
// hyprtk-persist partition the ISO's "with persistence" boot entry looks for.
//
// The package is deliberately split from the UI: all planning, guardrails and
// writing live here so they can be unit-tested without a real device.
package usb

import (
	"os/exec"
	"strings"
)

// Runner runs external commands. It exists so planning and guardrail logic can
// be tested against canned tool output instead of real hardware.
type Runner interface {
	// Output runs a command and returns its stdout.
	Output(name string, args ...string) ([]byte, error)
	// Run runs a command, feeding stdin (pass "" for none), and returns an
	// error on a non-zero exit.
	Run(stdin string, name string, args ...string) error
}

// ExecRunner runs commands on the host.
type ExecRunner struct{}

func (ExecRunner) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (ExecRunner) Run(stdin string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	return cmd.Run()
}

// Have reports whether a command is on PATH.
func Have(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
