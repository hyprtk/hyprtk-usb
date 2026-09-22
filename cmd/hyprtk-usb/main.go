// Command hyprtk-usb writes a hyprtk ISO to a USB stick and prepares the
// optional hyprtk-persist partition the ISO's "with persistence" boot entry
// looks for.
//
// Run with no arguments for an interactive TUI, or with flags for a
// non-interactive run (scriptable):
//
//	hyprtk-usb --iso ~/Documents/Isos/hyprtk-*.iso --target /dev/sda
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

const version = "0.1.0"

type options struct {
	iso     string
	target  string
	size    string
	noPers  bool
	refresh bool
	dryRun  bool
	yes     bool
	tui     bool
	test    bool
}

func main() { os.Exit(run(os.Args[1:])) }

func run(argv []string) int {
	fs := flag.NewFlagSet("hyprtk-usb", flag.ContinueOnError)
	fs.Usage = func() { usage(os.Stderr) }

	var o options
	var showVersion bool
	fs.StringVar(&o.iso, "iso", "", "ISO to write (default: newest hyprtk ISO)")
	fs.StringVar(&o.target, "target", "", "whole disk (e.g. /dev/sda); partitions are refused")
	fs.StringVar(&o.size, "size", "rest", "persistence size: 8G, 512M, 50%, or rest")
	fs.BoolVar(&o.noPers, "no-persist", false, "write the ISO only")
	fs.BoolVar(&o.refresh, "refresh", false, "keep an existing hyprtk-persist partition")
	fs.BoolVar(&o.dryRun, "dry-run", false, "print the plan; change nothing")
	fs.BoolVar(&o.yes, "y", false, "do not ask for confirmation")
	fs.BoolVar(&o.yes, "yes", false, "do not ask for confirmation (same as -y)")
	fs.BoolVar(&o.tui, "tui", false, "force the interactive TUI")
	fs.BoolVar(&o.test, "test", false, "relax checks to allow a regular-file target (testing)")
	fs.BoolVar(&showVersion, "version", false, "print the version and exit")

	if err := fs.Parse(argv); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if showVersion {
		fmt.Printf("hyprtk-usb %s\n", version)
		return 0
	}

	o.test = o.test || os.Getenv("HYPRTK_USB_TEST") == "1"

	// Writing needs root. Re-exec under sudo the same way the shell script did,
	// unless we are only planning or running a test.
	if !o.test && !o.dryRun && os.Geteuid() != 0 {
		return reexecSudo(argv)
	}

	if o.tui || len(argv) == 0 {
		return runTUI(o)
	}
	return runCLI(o)
}

// reexecSudo re-runs this program under sudo, preserving the arguments.
func reexecSudo(argv []string) int {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "hyprtk-usb: %v\n", err)
		return 1
	}
	cmd := exec.Command("sudo", append([]string{self}, argv...)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "hyprtk-usb: sudo: %v\n", err)
		return 1
	}
	return 0
}

func usage(w *os.File) {
	fmt.Fprint(w, `hyprtk-usb — write a hyprtk ISO to a USB stick (+ persistence)

Usage:
  hyprtk-usb                       interactive TUI
  hyprtk-usb --iso <iso> --target <dev> [flags]

Flags:
  --iso <file>        ISO to write (default: newest hyprtk ISO)
  --target <dev>      whole disk (e.g. /dev/sda); partitions are refused
  --size <spec>       persistence size: 8G, 512M, 50%, or rest (default)
  --no-persist        write the ISO only
  --refresh           keep an existing hyprtk-persist partition
  --dry-run           print the plan; change nothing
  -y, --yes           do not ask for confirmation
  --tui               force the interactive TUI
  --version           print the version
  -h, --help          show this help
`)
}
