package usb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeISO(t *testing.T, dir string) *ISO {
	t.Helper()
	path := filepath.Join(dir, "hyprtk.iso")
	if err := os.WriteFile(path, []byte("HYPRTK-ISO-CONTENTS"), 0o644); err != nil {
		t.Fatal(err)
	}
	iso, err := OpenISO(path, &fakeRunner{})
	if err != nil {
		t.Fatal(err)
	}
	return iso
}

func TestWriteFreshRunsSfdiskAndMkfs(t *testing.T) {
	dir := t.TempDir()
	iso := fakeISO(t, dir)
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	r := &fakeRunner{}
	plan := &Plan{
		ISO:          iso,
		TargetPath:   target,
		Mode:         ModeFresh,
		StartSectors: 2048,
		SizeSectors:  1000,
		PartitionDev: target + "3",
	}
	var stages []string
	if err := Write(iso, plan, r, func(p Progress) { stages = append(stages, p.Stage) }); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !r.called("sfdisk --append") {
		t.Error("expected an sfdisk --append call")
	}
	if !r.called("mkfs.ext4") {
		t.Error("expected an mkfs.ext4 call")
	}
	// The ISO bytes must have landed on the target.
	got, _ := os.ReadFile(target)
	if string(got) != "HYPRTK-ISO-CONTENTS" {
		t.Errorf("target contents = %q", got)
	}
	for _, want := range []string{"copy", "partition", "format", "done"} {
		if !contains(stages, want) {
			t.Errorf("stage %q missing from %v", want, stages)
		}
	}
}

func TestWriteNoPersistSkipsPartitioning(t *testing.T) {
	dir := t.TempDir()
	iso := fakeISO(t, dir)
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	r := &fakeRunner{}
	if err := Write(iso, &Plan{ISO: iso, TargetPath: target, Mode: ModeNoPersist}, r, nil); err != nil {
		t.Fatal(err)
	}
	if r.called("sfdisk") || r.called("mkfs.ext4") {
		t.Errorf("no-persist should not partition: %v", r.calls)
	}
}

func TestWriteRefreshSkipsMkfs(t *testing.T) {
	dir := t.TempDir()
	iso := fakeISO(t, dir)
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	r := &fakeRunner{}
	plan := &Plan{ISO: iso, TargetPath: target, Mode: ModeRefresh, StartSectors: 4096, SizeSectors: 8000, PartitionDev: target + "3"}
	if err := Write(iso, plan, r, nil); err != nil {
		t.Fatal(err)
	}
	if !r.called("sfdisk --append") {
		t.Error("refresh should re-append the partition")
	}
	if r.called("mkfs.ext4") {
		t.Error("refresh must not re-format the existing filesystem")
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if strings.EqualFold(x, want) {
			return true
		}
	}
	return false
}
