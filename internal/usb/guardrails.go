package usb

import (
	"fmt"
	"strings"
)

// ValidateInput is everything the guardrails need.
type ValidateInput struct {
	Dev      Device
	ISOSize  int64
	RootDisk string
	TestMode bool
}

// ValidateTarget refuses the obvious foot-guns. It never inspects the device
// beyond what ListDevices already read, so it is cheap and testable.
func ValidateTarget(in ValidateInput) error {
	if in.Dev.Path == "" {
		return fmt.Errorf("target: no device selected")
	}
	if in.Dev.Path == in.RootDisk && !in.TestMode {
		return fmt.Errorf("refusing to write the disk backing / (%s)", in.Dev.Path)
	}
	if in.Dev.Size <= in.ISOSize {
		return fmt.Errorf("target (%d bytes) is smaller than the ISO (%d bytes)", in.Dev.Size, in.ISOSize)
	}
	if m := in.Dev.Mounts(); len(m) > 0 {
		return fmt.Errorf("%s has mounted partitions (%s) - unmount them first",
			in.Dev.Path, strings.Join(m, ", "))
	}
	return nil
}
