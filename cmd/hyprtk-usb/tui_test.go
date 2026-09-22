package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hyprtk/hyprtk-usb/internal/usb"
)

type fakeRunner struct{}

func (fakeRunner) Output(name string, args ...string) ([]byte, error) { return nil, nil }
func (fakeRunner) Run(stdin, name string, args ...string) error       { return nil }

func press(t *testing.T, m model, name, text string) model {
	t.Helper()
	next, _ := m.handleKey(name, text)
	out, ok := next.(model)
	if !ok {
		t.Fatalf("handleKey returned %T", next)
	}
	return out
}

func TestTUIHappyPath(t *testing.T) {
	dir := t.TempDir()
	isoPath := filepath.Join(dir, "hyprtk-2026.09.22-x86_64.iso")
	if err := os.WriteFile(isoPath, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	m := model{
		st:      newStyles(defaultPalette()),
		r:       fakeRunner{},
		persist: true,
		isos:    []isoEntry{{path: isoPath, size: 4096}},
		devs:    []usb.Device{{Path: "/dev/sda", Size: 32 << 30, Model: "Ultra Fit"}},
	}

	m = press(t, m, "enter", "")
	if m.step != stepDevice {
		t.Fatalf("after ISO select step = %v, want device", m.step)
	}
	m = press(t, m, "enter", "")
	if m.step != stepOptions {
		t.Fatalf("after device select step = %v, want options", m.step)
	}

	m = press(t, m, "p", "")
	if m.persist {
		t.Error("p should toggle persistence off")
	}
	m = press(t, m, "p", "")
	m = press(t, m, "right", "")
	if sizeChoices[m.sizeIdx] != "8G" {
		t.Errorf("size = %q, want 8G", sizeChoices[m.sizeIdx])
	}

	m = press(t, m, "enter", "")
	if m.step != stepConfirm {
		t.Fatalf("after options step = %v (err %v), want confirm", m.step, m.err)
	}
	m = press(t, m, "x", "x")
	if m.step != stepConfirm {
		t.Error("a wrong confirmation must not proceed")
	}
}

func TestTUIDeviceStepHasNoWritableDisks(t *testing.T) {
	// The only device is the disk backing /, so none are writable.
	m := model{
		st:      newStyles(defaultPalette()),
		r:       rootDiskRunner{},
		persist: true,
		isos:    []isoEntry{{path: "x.iso", size: 4096}},
		devs:    []usb.Device{{Path: "/dev/nvme0n1", Size: 1 << 40}},
	}
	if got := m.writableDevices(); len(got) != 0 {
		t.Errorf("expected no writable devices, got %d", len(got))
	}
	m = press(t, m, "enter", "")
	if m.step != stepDevice {
		t.Fatalf("step = %v, want device", m.step)
	}
}

// rootDiskRunner makes RootDisk report the one device we pass in, so it is
// filtered out of writableDevices.
type rootDiskRunner struct{}

func (rootDiskRunner) Output(name string, args ...string) ([]byte, error) {
	switch name {
	case "findmnt":
		return []byte("/dev/nvme0n1p2\n"), nil
	case "lsblk":
		return []byte("nvme0n1\n"), nil
	}
	return nil, nil
}
func (rootDiskRunner) Run(stdin, name string, args ...string) error { return nil }
