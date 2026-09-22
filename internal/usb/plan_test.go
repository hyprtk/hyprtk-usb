package usb

import "testing"

func TestParseSize(t *testing.T) {
	avail := int64(1_000_000)
	cases := []struct {
		spec    string
		want    int64
		wantErr bool
	}{
		{"", avail, false},
		{"rest", avail, false},
		{"50%", avail * 50 / 100, false},
		{"1G", 1024 * 1024 * 1024 / 512, false},
		{"512M", 512 * 1024 * 1024 / 512, false},
		{"12345", 12345, false},
		{"0%", 0, true},
		{"101%", 0, true},
		{"1T", 0, true},
		{"abc", 0, true},
		{"-5", 0, true},
	}
	for _, c := range cases {
		got, err := parseSize(c.spec, avail)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseSize(%q) = %d, want error", c.spec, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("parseSize(%q) = %d, %v; want %d", c.spec, got, err, c.want)
		}
	}
}

func TestAlignUp(t *testing.T) {
	if got := alignUp(2049, 2048); got != 4096 {
		t.Errorf("alignUp(2049,2048) = %d, want 4096", got)
	}
	if got := alignUp(2048, 2048); got != 2048 {
		t.Errorf("alignUp(2048,2048) = %d, want 2048", got)
	}
}

func TestPartitionPath(t *testing.T) {
	cases := map[string]string{
		"/dev/sda":     "/dev/sda3",
		"/dev/nvme0n1": "/dev/nvme0n1p3",
		"/dev/loop0":   "/dev/loop0p3",
		"/dev/mmcblk0": "/dev/mmcblk0p3",
	}
	for dev, want := range cases {
		if got := PartitionPath(dev, 3); got != want {
			t.Errorf("PartitionPath(%q,3) = %q, want %q", dev, got, want)
		}
	}
}

func TestNextFreePartNum(t *testing.T) {
	dev := Device{Parts: []Partition{{Num: 1}, {Num: 2}}}
	if n, err := nextFreePartNum(dev); err != nil || n != 3 {
		t.Errorf("nextFreePartNum = %d, %v; want 3", n, err)
	}
	full := Device{Parts: []Partition{{Num: 1}, {Num: 2}, {Num: 3}, {Num: 4}}}
	if _, err := nextFreePartNum(full); err == nil {
		t.Error("expected an error when all four MBR slots are used")
	}
}

func isoOf(sectors int64) *ISO { return &ISO{Path: "x.iso", Size: sectors * 512} }

func TestBuildPlanNoPersist(t *testing.T) {
	p, err := BuildPlan(isoOf(1000), Device{Path: "/dev/sda", Size: 10_000 * 512}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != ModeNoPersist {
		t.Errorf("mode = %v, want none", p.Mode)
	}
}

func TestBuildPlanFresh(t *testing.T) {
	iso := isoOf(1000) // ends at sector 1000
	dev := Device{Path: "/dev/sda", Size: 100_000 * 512, Parts: []Partition{{Num: 1}, {Num: 2}}}
	p, err := BuildPlan(iso, dev, Options{Persist: true, SizeSpec: "rest"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != ModeFresh {
		t.Fatalf("mode = %v, want fresh", p.Mode)
	}
	if p.StartSectors != 2048 { // aligned up from 1000
		t.Errorf("start = %d, want 2048", p.StartSectors)
	}
	if want := int64((100_000 - 2048) / 2048 * 2048); p.SizeSectors != want {
		t.Errorf("size = %d, want %d", p.SizeSectors, want)
	}
	if p.PartitionNum != 3 || p.PartitionDev != "/dev/sda3" {
		t.Errorf("partition = %d %s, want 3 /dev/sda3", p.PartitionNum, p.PartitionDev)
	}
}

func TestBuildPlanRefresh(t *testing.T) {
	iso := isoOf(1000)
	dev := Device{Path: "/dev/sda", Size: 100_000 * 512, Parts: []Partition{
		{Num: 1}, {Num: 2},
		{Num: 3, Label: PersistLabel, Start: 4096 * 512, Size: 50_000 * 512, Path: "/dev/sda3"},
	}}
	p, err := BuildPlan(iso, dev, Options{Persist: true, Refresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != ModeRefresh || p.StartSectors != 4096 || p.SizeSectors != 50_000 {
		t.Fatalf("refresh plan wrong: %+v", p)
	}
}

func TestBuildPlanRefreshOverlap(t *testing.T) {
	iso := isoOf(100_000) // ISO extends past the existing partition start
	dev := Device{Path: "/dev/sda", Size: 200_000 * 512, Parts: []Partition{
		{Num: 1}, {Num: 2},
		{Num: 3, Label: PersistLabel, Start: 4096 * 512, Size: 50_000 * 512, Path: "/dev/sda3"},
	}}
	if _, err := BuildPlan(iso, dev, Options{Persist: true, Refresh: true}); err == nil {
		t.Error("expected an overlap error")
	}
}

func TestBuildPlanTooSmall(t *testing.T) {
	iso := isoOf(100_000)
	dev := Device{Path: "/dev/sda", Size: 100_500 * 512}
	if _, err := BuildPlan(iso, dev, Options{Persist: true}); err == nil {
		t.Error("expected a no-room error")
	}
}
