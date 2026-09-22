package usb

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// DefaultAlign is 1 MiB expressed in 512-byte sectors.
	DefaultAlign = int64(2048)
	// PersistLabel is the filesystem label the ISO's persistence entry matches.
	PersistLabel = "hyprtk-persist"
)

// Mode is what the writer will do about persistence.
type Mode int

const (
	ModeNoPersist Mode = iota // plain dd, no extra partition
	ModeFresh                 // create + format a new hyprtk-persist partition
	ModeRefresh               // re-attach the existing hyprtk-persist partition
)

func (m Mode) String() string {
	switch m {
	case ModeFresh:
		return "fresh"
	case ModeRefresh:
		return "refresh"
	default:
		return "none"
	}
}

// Options control how a plan is built.
type Options struct {
	Persist  bool
	SizeSpec string // "rest" (default), "8G", "512M", "50%", or a sector count
	Refresh  bool
	Align    int64
	TestMode bool // allow regular-file targets (integration tests)
}

// Plan is a fully-resolved, ready-to-execute write.
type Plan struct {
	ISO           *ISO
	TargetPath    string
	TargetSize    int64
	Mode          Mode
	StartSectors  int64
	SizeSectors   int64
	PartitionNum  int
	PartitionDev  string
	TestMode      bool  // skip mkfs on a regular-file target
	ExistingStart int64 // sectors (ModeRefresh)
	ExistingSize  int64 // sectors (ModeRefresh)
	Warnings      []string
}

// BuildPlan validates nothing about the target — that is ValidateTarget's job —
// and turns the ISO + device + options into concrete geometry.
func BuildPlan(iso *ISO, dev Device, opts Options) (*Plan, error) {
	align := opts.Align
	if align <= 0 {
		align = DefaultAlign
	}
	p := &Plan{ISO: iso, TargetPath: dev.Path, TargetSize: dev.Size, TestMode: opts.TestMode}

	if !opts.Persist {
		p.Mode = ModeNoPersist
		return p, nil
	}

	total := dev.Size / 512
	if existing, ok := dev.PersistPartition(); ok {
		start := existing.Start / 512
		size := existing.Size / 512
		if opts.Refresh {
			if start < iso.Sectors() {
				return nil, fmt.Errorf("existing %s partition overlaps the new ISO - back it up and write fresh", PersistLabel)
			}
			p.Mode = ModeRefresh
			p.StartSectors = start
			p.SizeSectors = size
			p.PartitionNum = existing.Num
			p.PartitionDev = existing.Path
			p.ExistingStart = start
			p.ExistingSize = size
			return p, nil
		}
		p.Warnings = append(p.Warnings,
			fmt.Sprintf("an existing %s partition will be replaced", PersistLabel))
	}

	p.Mode = ModeFresh
	p.StartSectors = alignUp(iso.Sectors(), align)
	if p.StartSectors > total-1 {
		return nil, fmt.Errorf("no room after the ISO for a persistence partition (%d sectors free)", total-p.StartSectors)
	}
	avail := total - p.StartSectors
	size, err := parseSize(opts.SizeSpec, avail)
	if err != nil {
		return nil, err
	}
	size = size / align * align
	if size < align {
		return nil, fmt.Errorf("persistence partition too small (need at least %d MiB)", align*512/1024/1024)
	}
	if size > avail {
		return nil, fmt.Errorf("persistence size exceeds the free space (%d MiB)", avail*512/1024/1024)
	}
	num, err := nextFreePartNum(dev)
	if err != nil {
		return nil, err
	}
	p.SizeSectors = size
	p.PartitionNum = num
	p.PartitionDev = PartitionPath(dev.Path, num)
	return p, nil
}

func alignUp(v, a int64) int64 {
	if a <= 1 {
		return v
	}
	return (v + a - 1) / a * a
}

// parseSize turns a size spec into a sector count. Availability bounds "rest"
// and percentage forms.
func parseSize(spec string, availSectors int64) (int64, error) {
	s := strings.TrimSpace(strings.ToLower(spec))
	switch {
	case s == "" || s == "rest":
		return availSectors, nil
	case strings.HasSuffix(s, "%"):
		pct, err := strconv.ParseInt(strings.TrimSuffix(s, "%"), 10, 64)
		if err != nil || pct <= 0 || pct > 100 {
			return 0, fmt.Errorf("bad size %q: use e.g. 8G, 512M, 50%% or rest", spec)
		}
		return availSectors * pct / 100, nil
	case strings.HasSuffix(s, "g"):
		n, err := strconv.ParseInt(strings.TrimSuffix(s, "g"), 10, 64)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("bad size %q: use e.g. 8G, 512M, 50%% or rest", spec)
		}
		return n * 1024 * 1024 * 1024 / 512, nil
	case strings.HasSuffix(s, "m"):
		n, err := strconv.ParseInt(strings.TrimSuffix(s, "m"), 10, 64)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("bad size %q: use e.g. 8G, 512M, 50%% or rest", spec)
		}
		return n * 1024 * 1024 / 512, nil
	default:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("bad size %q: use e.g. 8G, 512M, 50%% or rest", spec)
		}
		return n, nil // sectors
	}
}

// nextFreePartNum is the smallest MBR slot (1-4) not already in use.
func nextFreePartNum(dev Device) (int, error) {
	used := map[int]bool{}
	for _, p := range dev.Parts {
		used[p.Num] = true
	}
	for n := 1; n <= 4; n++ {
		if !used[n] {
			return n, nil
		}
	}
	return 0, fmt.Errorf("no free partition slot on %s (MBR allows 4)", dev.Path)
}

// PartitionPath returns the device path for partition n, handling the `p`
// infix used by nvme/mmcblk/loop devices.
func PartitionPath(devicePath string, n int) string {
	for _, prefix := range []string{"nvme", "mmcblk", "loop"} {
		if strings.Contains(devicePath, prefix) {
			return fmt.Sprintf("%sp%d", devicePath, n)
		}
	}
	return fmt.Sprintf("%s%d", devicePath, n)
}
