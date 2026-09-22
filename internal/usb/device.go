package usb

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Partition is one child of a block device.
type Partition struct {
	Path       string
	Size       int64 // bytes
	Start      int64 // bytes
	Label      string
	Mountpoint string
	Num        int
}

// Device is a whole-disk block device.
type Device struct {
	Path      string
	Name      string
	Size      int64 // bytes
	Model     string
	Tran      string
	Removable bool
	Parts     []Partition
}

// Mounts returns every mountpoint on the device, including its partitions'.
func (d Device) Mounts() []string {
	var out []string
	for _, p := range d.Parts {
		if p.Mountpoint != "" {
			out = append(out, p.Mountpoint)
		}
	}
	return out
}

// Describe is a one-line human summary for the UI.
func (d Device) Describe() string {
	what := strings.TrimSpace(d.Model)
	if what == "" {
		what = "disk"
	}
	where := d.Tran
	if where == "" {
		where = "?"
	}
	rm := ""
	if d.Removable {
		rm = ", removable"
	}
	return fmt.Sprintf("%s — %s, %s%s", d.Path, what, where, rm)
}

// PersistPartition returns the partition labelled hyprtk-persist, if any.
func (d Device) PersistPartition() (Partition, bool) {
	for _, p := range d.Parts {
		if p.Label == PersistLabel {
			return p, true
		}
	}
	return Partition{}, false
}

type lsblkNode struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	Size       int64       `json:"size"`
	Type       string      `json:"type"`
	Rm         bool        `json:"rm"`
	Tran       string      `json:"tran"`
	Model      string      `json:"model"`
	Mountpoint string      `json:"mountpoint"`
	Partn      int         `json:"partn"`
	Label      string      `json:"label"`
	Start      int64       `json:"start"`
	Children   []lsblkNode `json:"children"`
}

type lsblkOut struct {
	Blockdevices []lsblkNode `json:"blockdevices"`
}

const lsblkCols = "NAME,PATH,SIZE,TYPE,RM,TRAN,MODEL,MOUNTPOINT,PARTN,LABEL,START"

// ListDevices returns the host's whole-disk block devices and loop devices.
func ListDevices(r Runner) ([]Device, error) {
	out, err := r.Output("lsblk", "-bJ", "-o", lsblkCols)
	if err != nil {
		return nil, fmt.Errorf("lsblk: %w", err)
	}
	var parsed lsblkOut
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("lsblk: parse: %w", err)
	}

	var devs []Device
	for _, n := range parsed.Blockdevices {
		if n.Type != "disk" && n.Type != "loop" {
			continue
		}
		d := Device{
			Path:      n.Path,
			Name:      n.Name,
			Size:      n.Size,
			Model:     n.Model,
			Tran:      n.Tran,
			Removable: n.Rm,
		}
		for _, c := range n.Children {
			d.Parts = append(d.Parts, Partition{
				Path:       c.Path,
				Size:       c.Size,
				Start:      c.Start,
				Label:      c.Label,
				Mountpoint: c.Mountpoint,
				Num:        c.Partn,
			})
		}
		devs = append(devs, d)
	}
	return devs, nil
}

// FindDevice returns the device with the given path.
func FindDevice(r Runner, path string) (Device, error) {
	devs, err := ListDevices(r)
	if err != nil {
		return Device{}, err
	}
	for _, d := range devs {
		if d.Path == path {
			return d, nil
		}
	}
	return Device{}, fmt.Errorf("target: %s is not a whole-disk block device", path)
}

// RootDisk returns the whole disk that backs / (e.g. /dev/nvme0n1), or "" when
// it cannot be determined.
func RootDisk(r Runner) string {
	src, err := r.Output("findmnt", "-no", "SOURCE", "/")
	if err != nil {
		return ""
	}
	path := strings.TrimSpace(string(src))
	if path == "" {
		return ""
	}
	pk, err := r.Output("lsblk", "-no", "PKNAME", path)
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(pk))
	if name == "" {
		return ""
	}
	return "/dev/" + name
}
