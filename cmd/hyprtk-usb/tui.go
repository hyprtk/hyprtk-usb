package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/hyprtk/hyprtk-usb/internal/usb"
)

type step int

const (
	stepISO step = iota
	stepDevice
	stepOptions
	stepConfirm
	stepRun
	stepDone
	stepError
)

type isoEntry struct {
	path  string
	label string
	size  int64
}

var sizeChoices = []string{"rest", "8G", "4G", "2G"}

type model struct {
	st styles
	r  usb.Runner
	o  options

	step   step
	isos   []isoEntry
	isoIdx int
	devs   []usb.Device
	devIdx int

	sizeIdx int
	refresh bool
	persist bool

	confirm string
	plan    *usb.Plan
	prog    usb.Progress
	err     error
	runCh   chan runMsg
}

type runMsg struct {
	prog usb.Progress
	err  error
	done bool
}

func runTUI(o options) int {
	p := loadPalette()
	m := newModel(o, p)
	final, err := tea.NewProgram(m, tea.WithInput(ttyInput())).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "hyprtk-usb: %v\n", err)
		return 1
	}
	if res, ok := final.(model); ok && res.err != nil {
		return 1
	}
	return 0
}

func ttyInput() io.Reader {
	f, err := os.Open("/dev/tty")
	if err != nil {
		return os.Stdin
	}
	return f
}

func newModel(o options, p Palette) model {
	m := model{
		st:      newStyles(p),
		r:       usb.ExecRunner{},
		o:       o,
		persist: true,
		sizeIdx: 0,
	}

	for _, path := range scanISOs() {
		iso, err := usb.OpenISO(path, m.r)
		if err != nil {
			continue
		}
		m.isos = append(m.isos, isoEntry{path: path, label: iso.Label, size: iso.Size})
	}
	if len(m.isos) == 0 {
		m.step = stepError
		m.err = fmt.Errorf("no hyprtk ISO found in ~/Documents/Isos or ~")
		return m
	}
	devs, err := usb.ListDevices(m.r)
	if err != nil {
		m.step = stepError
		m.err = err
		return m
	}
	m.devs = devs
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) selectedISO() isoEntry { return m.isos[m.isoIdx] }

func (m model) writableDevices() []usb.Device {
	if len(m.isos) == 0 {
		return nil
	}
	isoSize := m.selectedISO().size
	root := usb.RootDisk(m.r)
	var out []usb.Device
	for _, d := range m.devs {
		if usb.ValidateTarget(usb.ValidateInput{Dev: d, ISOSize: isoSize, RootDisk: root, TestMode: m.o.test}) == nil {
			out = append(out, d)
		}
	}
	return out
}

func (m model) selectedDevice() (usb.Device, bool) {
	devs := m.writableDevices()
	if m.devIdx < 0 || m.devIdx >= len(devs) {
		return usb.Device{}, false
	}
	return devs[m.devIdx], true
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.updateKey(msg)
	case runMsg:
		return m.updateRun(msg)
	}
	return m, nil
}

func (m model) updateKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	return m.handleKey(key.String(), key.Text)
}

// handleKey is separated from the tea message so the flow can be unit-tested.
func (m model) handleKey(name, text string) (tea.Model, tea.Cmd) {
	if name == "ctrl+c" {
		return m, tea.Quit
	}
	if name == "esc" && m.step != stepRun {
		return m, tea.Quit
	}

	switch m.step {
	case stepISO:
		switch name {
		case "up", "k":
			m.isoIdx = clampIdx(m.isoIdx-1, len(m.isos))
		case "down", "j":
			m.isoIdx = clampIdx(m.isoIdx+1, len(m.isos))
		case "enter":
			m.devIdx = 0
			m.step = stepDevice
		}
	case stepDevice:
		devs := m.writableDevices()
		switch name {
		case "up", "k":
			m.devIdx = clampIdx(m.devIdx-1, len(devs))
		case "down", "j":
			m.devIdx = clampIdx(m.devIdx+1, len(devs))
		case "enter":
			if len(devs) == 0 {
				break
			}
			m.refresh = false
			if _, ok := devs[m.devIdx].PersistPartition(); ok {
				m.refresh = m.persist
			}
			m.step = stepOptions
		}
	case stepOptions:
		switch name {
		case "p":
			m.persist = !m.persist
		case "r":
			m.refresh = !m.refresh
		case "left", "h":
			m.sizeIdx = clampIdx(m.sizeIdx-1, len(sizeChoices))
		case "right", "l", "tab":
			m.sizeIdx = clampIdx(m.sizeIdx+1, len(sizeChoices))
		case "enter":
			if err := m.buildPlan(); err != nil {
				m.step = stepError
				m.err = err
				return m, nil
			}
			m.confirm = ""
			m.step = stepConfirm
		}
	case stepConfirm:
		dev, _ := m.selectedDevice()
		switch name {
		case "backspace":
			if m.confirm != "" {
				m.confirm = m.confirm[:len(m.confirm)-1]
			}
		case "enter":
			if m.confirm == dev.Path {
				m.step = stepRun
				return m, m.startRun()
			}
		default:
			if text != "" {
				m.confirm += text
			}
		}
	}
	return m, nil
}

func (m *model) buildPlan() error {
	dev, ok := m.selectedDevice()
	if !ok {
		return fmt.Errorf("no target device selected")
	}
	iso, err := usb.OpenISO(m.selectedISO().path, m.r)
	if err != nil {
		return err
	}
	plan, err := buildValidatedPlan(iso, dev, options{
		noPers:  !m.persist,
		size:    sizeChoices[m.sizeIdx],
		refresh: m.refresh,
		test:    m.o.test,
	}, m.r)
	if err != nil {
		return err
	}
	m.plan = plan
	return nil
}

func (m *model) startRun() tea.Cmd {
	iso, err := usb.OpenISO(m.selectedISO().path, m.r)
	if err != nil {
		m.err = err
		return tea.Quit
	}
	ch := make(chan runMsg, 64)
	m.runCh = ch
	go func() {
		err := usb.Write(iso, m.plan, m.r, func(p usb.Progress) { ch <- runMsg{prog: p} })
		ch <- runMsg{done: true, err: err}
		close(ch)
	}()
	return waitRun(ch)
}

func waitRun(ch chan runMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return runMsg{done: true}
		}
		return msg
	}
}

func (m model) updateRun(msg runMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.step = stepError
		return m, nil
	}
	if msg.done {
		m.step = stepDone
		return m, nil
	}
	m.prog = msg.prog
	return m, waitRun(m.runCh)
}

func (m model) View() tea.View {
	var b strings.Builder
	b.WriteString(m.st.header())
	switch m.step {
	case stepISO:
		b.WriteString("\n  Select the ISO to write:\n\n")
		items := make([]string, len(m.isos))
		for i, e := range m.isos {
			label := e.label
			if label == "" {
				label = "?"
			}
			items[i] = fmt.Sprintf("%-40s  %-14s  %s", e.path, label, humanBytes(e.size))
		}
		b.WriteString(renderList(m.st, items, m.isoIdx, "no ISOs found"))
		b.WriteString(m.st.footer("up/down move  ·  enter select  ·  esc quit"))
	case stepDevice:
		b.WriteString("\n  Select the target disk (whole disks only):\n\n")
		devs := m.writableDevices()
		items := make([]string, len(devs))
		for i, d := range devs {
			items[i] = fmt.Sprintf("%-14s  %-28s  %s", d.Path, d.Model, humanBytes(d.Size))
		}
		b.WriteString(renderList(m.st, items, m.devIdx, "no writable disks found"))
		b.WriteString(m.st.footer("up/down move  ·  enter select  ·  esc quit"))
	case stepOptions:
		dev, _ := m.selectedDevice()
		b.WriteString(fmt.Sprintf("\n  Options for %s:\n\n", dev.Path))
		b.WriteString(fmt.Sprintf("  %s  persistence        %s\n",
			checkbox(m.st, m.persist), onOff(m.st, m.persist)))
		if m.persist {
			b.WriteString(fmt.Sprintf("  %s  size               %s\n",
				m.st.accent.Render("[s]"), sizeChoices[m.sizeIdx]))
			if _, ok := dev.PersistPartition(); ok {
				b.WriteString(fmt.Sprintf("  %s  keep existing      %s\n",
					m.st.accent.Render("[r]"), onOff(m.st, m.refresh)))
			}
		}
		b.WriteString(m.st.footer("p toggle  ·  s/left/right size  ·  r keep  ·  enter continue  ·  esc quit"))
	case stepConfirm:
		dev, _ := m.selectedDevice()
		b.WriteString(m.st.err.Render(fmt.Sprintf("\n  This ERASES %s.", dev.Path)) + "\n\n")
		b.WriteString("  Type the device path to confirm:\n\n")
		b.WriteString("  > " + m.st.accent.Render(m.confirm) + "\n")
		b.WriteString(m.st.footer("type the path  ·  enter start  ·  esc quit"))
	case stepRun:
		b.WriteString("\n" + m.progressView())
		b.WriteString(m.st.footer("working - do not unplug the device"))
	case stepDone:
		b.WriteString(m.st.ok.Render("\n  done.") + "\n\n")
		b.WriteString("  Boot the stick and pick \"Hyprtk live with persistence\".\n")
		b.WriteString(m.st.footer("enter quit"))
	case stepError:
		msg := "unknown error"
		if m.err != nil {
			msg = m.err.Error()
		}
		b.WriteString(m.st.err.Render("\n  error: "+msg) + "\n")
		b.WriteString(m.st.footer("esc quit"))
	}
	return tea.NewView(b.String())
}

func (m model) progressView() string {
	switch m.prog.Stage {
	case "copy":
		if m.prog.Total == 0 {
			return "  copying...\n"
		}
		pct := float64(m.prog.Written) / float64(m.prog.Total) * 100
		return fmt.Sprintf("  copying  %5.1f%%  %s / %s\n%s\n",
			pct, humanBytes(m.prog.Written), humanBytes(m.prog.Total), bar(m.st, pct, 40))
	case "partition":
		return "  adding the persistence partition...\n"
	case "format":
		return "  formatting " + usb.PersistLabel + "...\n"
	default:
		return "  finishing...\n"
	}
}

func renderList(st styles, items []string, idx int, empty string) string {
	if len(items) == 0 {
		return st.warn.Render("  "+empty) + "\n"
	}
	var b strings.Builder
	for i, it := range items {
		if i == idx {
			b.WriteString(st.selected.Render("  " + it))
		} else {
			b.WriteString("  " + it)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func (st styles) header() string {
	return st.title.Render("  hyprtk-usb") + "\n" +
		st.dim.Render("  write a hyprtk ISO to a USB stick (+ persistence)") + "\n"
}

func (st styles) footer(help string) string {
	return "\n" + st.help.Render("  "+help) + "\n"
}

func checkbox(st styles, on bool) string {
	if on {
		return st.accent.Render("[x]")
	}
	return st.dim.Render("[ ]")
}

func onOff(st styles, on bool) string {
	if on {
		return st.ok.Render("on")
	}
	return st.dim.Render("off")
}

func bar(st styles, pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return "  " + st.accent.Render(strings.Repeat("#", filled)) +
		st.dim.Render(strings.Repeat("-", width-filled))
}

func clampIdx(i, n int) int {
	if n == 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}
