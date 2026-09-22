package usb

import (
	"fmt"
	"os"
	"strings"
)

// ISO is a hyprtk ISO image on disk.
type ISO struct {
	Path  string
	Size  int64
	Label string
}

// OpenISO stats an ISO and reads its volume label (best-effort: a missing
// blkid just leaves the label empty).
func OpenISO(path string, r Runner) (*ISO, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("iso: %w", err)
	}
	if st.IsDir() {
		return nil, fmt.Errorf("iso: %s is a directory", path)
	}
	iso := &ISO{Path: path, Size: st.Size()}
	if out, err := r.Output("blkid", "-p", "-o", "value", "-s", "LABEL", path); err == nil {
		iso.Label = strings.TrimSpace(string(out))
	}
	return iso, nil
}

// Sectors is the ISO size in 512-byte sectors, rounded up.
func (i *ISO) Sectors() int64 { return (i.Size + 511) / 512 }

// LooksHyprtk reports whether the volume label matches the hyprtk ISOs.
func (i *ISO) LooksHyprtk() bool { return strings.HasPrefix(i.Label, "HYPRTK") }
