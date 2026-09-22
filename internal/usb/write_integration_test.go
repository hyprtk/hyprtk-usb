package usb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAppendPartitionIntegration exercises the real sfdisk against a regular
// file that carries an isohybrid-like MBR, and checks that the two existing
// entries survive and a Linux partition is appended. Skips when sfdisk is
// absent or -short is set.
func TestAppendPartitionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	if !Have("sfdisk") {
		t.Skip("sfdisk not installed")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "stick.img")
	f, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(64 << 20); err != nil { // 64 MiB sparse
		t.Fatal(err)
	}
	f.Close()

	// A two-entry table within the first ~16 MiB, mirroring an archiso ISO.
	r := &ExecRunner{}
	craft := "label: dos\nunit: sectors\n\nstart=64, size=20000, type=0, bootable\nstart=20064, size=10000, type=ef\n"
	if err := r.Run(craft, "sfdisk", target); err != nil {
		t.Fatalf("sfdisk craft: %v", err)
	}

	total := int64(64<<20) / 512
	start := int64(32768) // 16 MiB, 1 MiB-aligned
	size := total - start
	if err := appendPartition(r, target, start, size); err != nil {
		t.Fatalf("appendPartition: %v", err)
	}

	out, err := r.Output("sfdisk", "-d", target)
	if err != nil {
		t.Fatalf("sfdisk -d: %v", err)
	}
	// Normalise whitespace (sfdisk pads the numbers) so the checks are exact.
	dump := strings.Join(strings.Fields(string(out)), "")
	for _, want := range []string{
		"start=64,size=20000,type=0,bootable",
		"start=20064,size=10000,type=ef",
		"start=32768",
		"type=83",
	} {
		if !strings.Contains(dump, want) {
			t.Errorf("sfdisk dump missing %q:\n%s", want, dump)
		}
	}
}
