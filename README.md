# hyprtk-usb

Write a [hyprtk](https://github.com/hyprtk/Hyprtk-ISO-Creator) ISO to a USB
stick and, optionally, add the **`hyprtk-persist`** partition the ISO's
*Hyprtk live with persistence* boot entry looks for.

It is the Go successor to the `hyprtk-usb` shell script: a single static binary,
the same guardrails, plus a TUI with a device picker, live `dd` progress and an
ISO check.

There are two implementations with the same behaviour: a **Go** binary (this
repo's root — static, released) and a peer **Python** one in [`python/`](python/)
that renders the hyprtk look with `rich` (double-border panels, mauve/cyan).

## Usage

```bash
# interactive TUI
hyprtk-usb

# non-interactive (scriptable)
sudo hyprtk-usb --iso ~/Documents/Isos/hyprtk-*.iso --target /dev/sda
```

Run with no arguments for the TUI; pass any flag for a non-interactive run.
Writing needs root — the program re-execs itself under `sudo` (the TUI does too).

### Flags

| Flag | Meaning |
| --- | --- |
| `--iso <file>` | ISO to write (default: newest `hyprtk-*.iso` in `~/Documents/Isos` or `~`) |
| `--target <dev>` | whole disk (e.g. `/dev/sda`); partitions are refused |
| `--size <spec>` | persistence size: `8G`, `512M`, `50%`, or `rest` (default) |
| `--no-persist` | write the ISO only |
| `--refresh` | re-write the ISO, keeping an existing `hyprtk-persist` partition |
| `--dry-run` | print the plan; change nothing |
| `-y`, `--yes` | do not ask for confirmation |
| `--tui` | force the interactive TUI |
| `--version` | print the version |

## How it works

1. `dd` the ISO (iso-hybrid, MBR) to the whole disk.
2. Append a **1 MiB-aligned** Linux partition in the free space with
   `sfdisk --append`, preserving the ISO's own two MBR entries (the iso9660 and
   the EFI FAT).
3. `mkfs.ext4 -L hyprtk-persist` on the new partition.

The ISO's *Hyprtk live with persistence* boot entry passes
`cow_label=hyprtk-persist`, so an archiso overlay `upperdir`/`workdir` is created
on that filesystem. Booting the default entry stays ephemeral.

With `--refresh`, an existing `hyprtk-persist` partition's exact geometry is read
before the `dd` and re-created unchanged afterwards, so the filesystem is never
resized (and the ISO must not overlap it).

## Safety

The writer refuses: a partition instead of a whole disk, a disk with mounted
partitions, the disk backing `/`, a target smaller than the ISO, an out-of-range
size, and (non-fatally, as a warning) an ISO that does not look like hyprtk.
Non-interactive runs require the exact device path typed to confirm unless `-y`.

## Build

```bash
make build      # bin/hyprtk-usb (static, CGO_ENABLED=0)
make test
make install    # ~/.local/bin/hyprtk-usb
```

Requires `dd`, `sfdisk`, `mkfs.ext4`, `blkid`, `lsblk` and `findmnt` at runtime
(the Go binary shells out to them).

## Layout

```
cmd/hyprtk-usb/    flags, TUI, non-interactive CLI
internal/usb/      ISO + device discovery, planning, guardrails, writer (+ tests)
python/            peer Python implementation (rich UI)
```
