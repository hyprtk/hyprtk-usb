package usb

import (
	"fmt"
	"io"
	"os"
)

// Progress is a step update. Stage is one of:
// "copy" (Written/Total bytes), "partition", "format", "done".
type Progress struct {
	Stage   string
	Written int64
	Total   int64
}

// Write executes a plan: copy the ISO to the target, then (unless the plan says
// otherwise) append the persistence partition and format it.
func Write(iso *ISO, plan *Plan, r Runner, onProgress func(Progress)) error {
	report := func(p Progress) {
		if onProgress != nil {
			onProgress(p)
		}
	}

	report(Progress{Stage: "copy", Total: iso.Size})
	if err := copyISO(iso.Path, plan.TargetPath, iso.Size, func(n int64) {
		report(Progress{Stage: "copy", Written: n, Total: iso.Size})
	}); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	// Ask the kernel to re-read the table; harmless if it is a regular file.
	_ = r.Run("", "blockdev", "--rereadpt", plan.TargetPath)

	if plan.Mode == ModeNoPersist {
		report(Progress{Stage: "done"})
		return nil
	}

	report(Progress{Stage: "partition"})
	if err := appendPartition(r, plan.TargetPath, plan.StartSectors, plan.SizeSectors); err != nil {
		return fmt.Errorf("sfdisk: %w", err)
	}
	_ = r.Run("", "blockdev", "--rereadpt", plan.TargetPath)

	if plan.Mode == ModeFresh && !plan.TestMode {
		report(Progress{Stage: "format"})
		if err := r.Run("", "mkfs.ext4", "-F", "-L", PersistLabel, plan.PartitionDev); err != nil {
			return fmt.Errorf("mkfs.ext4: %w", err)
		}
	}

	report(Progress{Stage: "done"})
	return nil
}

// appendPartition appends a Linux partition to the MBR, preserving existing
// entries (sfdisk reads and rewrites the table in place).
func appendPartition(r Runner, target string, start, size int64) error {
	line := fmt.Sprintf("start=%d, size=%d, type=83\n", start, size)
	return r.Run(line, "sfdisk", "--append", target)
}

// copyISO streams the ISO onto the target. Opening without O_TRUNC means a
// block device is written in place; a regular file keeps its length.
func copyISO(srcPath, dstPath string, total int64, onWritten func(int64)) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()

	buf := make([]byte, 4<<20)
	var written int64
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if onWritten != nil {
				onWritten(written)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	return dst.Sync()
}
