package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hyprtk/hyprtk-usb/internal/usb"
)

// scanISOs returns hyprtk ISOs newest-first, looking in ~/Documents/Isos and ~.
func scanISOs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	type entry struct {
		path string
		mod  time.Time
	}
	var found []entry
	for _, dir := range []string{filepath.Join(home, "Documents", "Isos"), home} {
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			n := e.Name()
			if e.IsDir() || !strings.HasPrefix(n, "hyprtk-") || !strings.HasSuffix(n, ".iso") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			found = append(found, entry{filepath.Join(dir, n), info.ModTime()})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].mod.After(found[j].mod) })
	paths := make([]string, len(found))
	for i, e := range found {
		paths[i] = e.path
	}
	return paths
}

func resolveISO(explicit string, r usb.Runner) (*usb.ISO, error) {
	if explicit != "" {
		return usb.OpenISO(explicit, r)
	}
	isos := scanISOs()
	if len(isos) == 0 {
		return nil, fmt.Errorf("no hyprtk ISO found - pass --iso <file>")
	}
	return usb.OpenISO(isos[0], r)
}

func runCLI(o options) int {
	r := usb.ExecRunner{}
	st := newStyles(loadPalette())

	iso, err := resolveISO(o.iso, r)
	if err != nil {
		return fail(err)
	}
	if !iso.LooksHyprtk() {
		fmt.Fprintf(os.Stderr, "hyprtk-usb: warning: %s does not look like a hyprtk ISO (label %q)\n", iso.Path, iso.Label)
	}
	if o.target == "" {
		return fail(fmt.Errorf("no --target device given"))
	}
	dev, err := resolveTargetDevice(r, o.target, o.test)
	if err != nil {
		return fail(err)
	}
	plan, err := buildValidatedPlan(iso, dev, o, r)
	if err != nil {
		return fail(err)
	}

	printPlan(st, iso, dev, plan)
	if o.dryRun {
		fmt.Println(st.warn.Render("  dry run - nothing written"))
		return 0
	}
	if !o.yes {
		if !confirmTyped(dev.Path) {
			fmt.Println("Aborted.")
			return 0
		}
	}
	if err := usb.Write(iso, plan, r, progressPrinter(st)); err != nil {
		return fail(err)
	}
	fmt.Println("\n" + st.ok.Render("  done - boot the stick and pick \"Hyprtk live with persistence\""))
	if plan.Mode == usb.ModeNoPersist {
		fmt.Println(st.dim.Render("  (no persistence partition; the default boot entry is fine)"))
	}
	return 0
}

// resolveTargetDevice finds the target block device, or — in test mode only —
// accepts a regular file so the whole flow can be exercised without hardware.
func resolveTargetDevice(r usb.Runner, target string, test bool) (usb.Device, error) {
	if test {
		if st, err := os.Stat(target); err == nil && st.Mode().IsRegular() {
			return usb.Device{Path: target, Name: filepath.Base(target), Size: st.Size()}, nil
		}
	}
	return usb.FindDevice(r, target)
}

// buildValidatedPlan runs the shared guardrails + planning for both the CLI and
// the TUI.
func buildValidatedPlan(iso *usb.ISO, dev usb.Device, o options, r usb.Runner) (*usb.Plan, error) {
	root := usb.RootDisk(r)
	if err := usb.ValidateTarget(usb.ValidateInput{
		Dev:      dev,
		ISOSize:  iso.Size,
		RootDisk: root,
		TestMode: o.test,
	}); err != nil {
		return nil, err
	}
	return usb.BuildPlan(iso, dev, usb.Options{
		Persist:  !o.noPers,
		SizeSpec: o.size,
		Refresh:  o.refresh,
		TestMode: o.test,
	})
}

func fail(err error) int {
	fmt.Fprintf(os.Stderr, "hyprtk-usb: %v\n", err)
	return 1
}

func printPlan(st styles, iso *usb.ISO, dev usb.Device, plan *usb.Plan) {
	fmt.Println(st.title.Render("  hyprtk-usb"))
	fmt.Printf("  ISO:    %s (%s)\n", iso.Path, humanBytes(iso.Size))
	if iso.Label != "" {
		fmt.Printf("  Label:  %s\n", iso.Label)
	}
	fmt.Printf("  Target: %s\n", dev.Describe())
	fmt.Printf("  Size:   %s\n", humanBytes(dev.Size))
	switch plan.Mode {
	case usb.ModeNoPersist:
		fmt.Println("  Persist: none")
	case usb.ModeRefresh:
		fmt.Printf("  Persist: %s (sectors %d..%d, %s) - existing, kept\n",
			plan.PartitionDev, plan.StartSectors, plan.StartSectors+plan.SizeSectors-1, humanBytes(plan.SizeSectors*512))
	default:
		fmt.Printf("  Persist: %s (sectors %d..%d, %s) label %s\n",
			plan.PartitionDev, plan.StartSectors, plan.StartSectors+plan.SizeSectors-1,
			humanBytes(plan.SizeSectors*512), usb.PersistLabel)
	}
	for _, w := range plan.Warnings {
		fmt.Println(st.warn.Render("  !  " + w))
	}
}

func confirmTyped(target string) bool {
	fmt.Printf("  This ERASES %s. Type the device path to confirm:\n  > ", target)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line) == target
}

// progressPrinter renders dd progress on one line, then a line per stage.
func progressPrinter(st styles) func(usb.Progress) {
	tty := isTTY()
	last := ""
	return func(p usb.Progress) {
		switch p.Stage {
		case "copy":
			if p.Total == 0 {
				return
			}
			pct := float64(p.Written) / float64(p.Total) * 100
			line := fmt.Sprintf("  copying %5.1f%%  %s / %s", pct, humanBytes(p.Written), humanBytes(p.Total))
			if tty {
				fmt.Printf("\r%s", line)
			}
			last = line
		case "partition":
			endLine(tty, last)
			fmt.Println(st.accent.Render("  adding the persistence partition..."))
		case "format":
			endLine(tty, last)
			fmt.Println(st.accent.Render("  formatting " + usb.PersistLabel + "..."))
		case "done":
			endLine(tty, last)
		}
	}
}

func endLine(tty bool, line string) {
	if tty && line != "" {
		fmt.Println()
	}
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
