package usb

import (
	"strings"
	"testing"
)

// fakeRunner records commands and returns canned output.
type fakeRunner struct {
	outFn func(name string, args ...string) ([]byte, error)
	runFn func(stdin, name string, args ...string) error
	calls []string
}

func (f *fakeRunner) Output(name string, args ...string) ([]byte, error) {
	if f.outFn != nil {
		return f.outFn(name, args...)
	}
	return nil, nil
}

func (f *fakeRunner) Run(stdin, name string, args ...string) error {
	f.calls = append(f.calls, name+" "+strings.Join(args, " ")+" | "+strings.TrimSpace(stdin))
	if f.runFn != nil {
		return f.runFn(stdin, name, args...)
	}
	return nil
}

func (f *fakeRunner) called(substr string) bool {
	for _, c := range f.calls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

const lsblkSample = `{
  "blockdevices": [
    {"name":"sda","path":"/dev/sda","size":8589934592,"type":"disk","rm":true,"tran":"usb","model":"Ultra Fit","mountpoint":null,"partn":null,"label":null,"start":null,
     "children":[
       {"name":"sda1","path":"/dev/sda1","size":2345678,"type":"part","partn":1,"mountpoint":null,"label":"HYPRTK_202609","start":32768},
       {"name":"sda2","path":"/dev/sda2","size":273684480,"type":"part","partn":2,"mountpoint":"/run/media/x/EFI","label":"EFI","start":2376496}
     ]},
    {"name":"nvme0n1","path":"/dev/nvme0n1","size":1000204886016,"type":"disk","rm":false,"tran":"nvme","model":"Samsung","mountpoint":null,"partn":null,"label":null,"start":null,
     "children":[
       {"name":"nvme0n1p1","path":"/dev/nvme0n1p1","size":536870912,"type":"part","partn":1,"mountpoint":"/boot","label":null,"start":2048},
       {"name":"nvme0n1p2","path":"/dev/nvme0n1p2","size":999000000000,"type":"part","partn":2,"mountpoint":"/","label":null,"start":1050624}
     ]},
    {"name":"loop0","path":"/dev/loop0","size":67108864,"type":"loop","rm":false,"tran":null,"model":null,"mountpoint":null,"partn":null,"label":null,"start":null,"children":[]}
  ]
}`

func TestListDevices(t *testing.T) {
	r := &fakeRunner{outFn: func(name string, args ...string) ([]byte, error) {
		if name == "lsblk" {
			return []byte(lsblkSample), nil
		}
		return nil, nil
	}}
	devs, err := ListDevices(r)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devs) != 3 {
		t.Fatalf("want 3 devices (2 disks + loop), got %d", len(devs))
	}
	sda := devs[0]
	if sda.Path != "/dev/sda" || !sda.Removable || sda.Tran != "usb" {
		t.Errorf("sda fields wrong: %+v", sda)
	}
	if len(sda.Parts) != 2 || sda.Parts[0].Num != 1 || sda.Parts[1].Mountpoint != "/run/media/x/EFI" {
		t.Errorf("sda partitions wrong: %+v", sda.Parts)
	}
	if got := sda.Mounts(); len(got) != 1 || got[0] != "/run/media/x/EFI" {
		t.Errorf("sda mounts = %v", got)
	}
}

func TestPersistPartition(t *testing.T) {
	dev := Device{Parts: []Partition{
		{Num: 1, Label: "HYPRTK_202609"},
		{Num: 3, Label: PersistLabel, Start: 100 << 20, Size: 50 << 20},
	}}
	p, ok := dev.PersistPartition()
	if !ok || p.Num != 3 {
		t.Fatalf("PersistPartition = %+v, %v", p, ok)
	}
	if _, ok := (Device{}).PersistPartition(); ok {
		t.Error("empty device reported a persist partition")
	}
}

func TestRootDisk(t *testing.T) {
	r := &fakeRunner{outFn: func(name string, args ...string) ([]byte, error) {
		switch name {
		case "findmnt":
			return []byte("/dev/nvme0n1p2\n"), nil
		case "lsblk":
			return []byte("nvme0n1\n"), nil
		}
		return nil, nil
	}}
	if got := RootDisk(r); got != "/dev/nvme0n1" {
		t.Errorf("RootDisk = %q, want /dev/nvme0n1", got)
	}
}

func TestFindDeviceRejectsPartition(t *testing.T) {
	r := &fakeRunner{outFn: func(name string, args ...string) ([]byte, error) {
		return []byte(lsblkSample), nil
	}}
	if _, err := FindDevice(r, "/dev/sda1"); err == nil {
		t.Error("FindDevice accepted a partition")
	}
	if _, err := FindDevice(r, "/dev/sda"); err != nil {
		t.Errorf("FindDevice rejected a whole disk: %v", err)
	}
}
