package usb

import "testing"

func TestValidateTarget(t *testing.T) {
	base := ValidateInput{
		Dev:      Device{Path: "/dev/sda", Size: 1000},
		ISOSize:  100,
		RootDisk: "/dev/nvme0n1",
	}
	if err := ValidateTarget(base); err != nil {
		t.Fatalf("valid target rejected: %v", err)
	}

	t.Run("no device", func(t *testing.T) {
		in := base
		in.Dev.Path = ""
		if err := ValidateTarget(in); err == nil {
			t.Error("expected an error")
		}
	})
	t.Run("mounted", func(t *testing.T) {
		in := base
		in.Dev.Parts = []Partition{{Mountpoint: "/run/media/x/EFI"}}
		if err := ValidateTarget(in); err == nil {
			t.Error("expected an error")
		}
	})
	t.Run("root disk", func(t *testing.T) {
		in := base
		in.Dev.Path = "/dev/nvme0n1"
		if err := ValidateTarget(in); err == nil {
			t.Error("expected an error")
		}
	})
	t.Run("too small", func(t *testing.T) {
		in := base
		in.Dev.Size = 50
		if err := ValidateTarget(in); err == nil {
			t.Error("expected an error")
		}
	})
	t.Run("root allowed in test mode", func(t *testing.T) {
		in := base
		in.Dev.Path = "/dev/nvme0n1"
		in.RootDisk = "/dev/nvme0n1"
		in.TestMode = true
		if err := ValidateTarget(in); err != nil {
			t.Errorf("test mode should relax the root-disk guard: %v", err)
		}
	})
}
