# hyprtk-usb

Write a [hyprtk](https://github.com/hyprtk/Hyprtk-ISO-Creator) ISO to a USB
stick and, optionally, add the **`hyprtk-persist`** partition the ISO's
*Hyprtk live with persistence* boot entry looks for.

A Python app using [`rich`](https://github.com/Textualize/rich), carrying the
hyprtk look: **double-border panels** with the **mauve (`#c084fc`) / cyan
(`#22d3ee`)** palette resolved from pywal, as **inline prompts** — no full-screen
takeover.

## Install

```bash
# single-file (needs Python 3.10+); from a GitHub release
python hyprtk-usb.pyz

# or from a wheel
pip install hyprtk_usb-*.whl

# or from source
cd python && pip install -e .

# or run in place
cd python && PYTHONPATH=. python -m hyprtk_usb
```

## Usage

```bash
# interactive (TUI)
hyprtk-usb

# non-interactive (scriptable)
sudo hyprtk-usb --iso ~/Documents/Isos/hyprtk-*.iso --target /dev/sda
```

Run with no flags for the interactive flow; pass any flag for a non-interactive
run. Writing needs root — the program re-execs itself under `sudo`.

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

1. Stream the ISO (iso-hybrid, MBR) onto the whole disk.
2. Append a **1 MiB-aligned** Linux partition in the free space with
   `sfdisk --append`, preserving the ISO's own two MBR entries (the iso9660 and
   the EFI FAT).
3. `mkfs.ext4 -L hyprtk-persist` on the new partition.

The ISO's *Hyprtk live with persistence* boot entry passes
`cow_label=hyprtk-persist`, so an archiso overlay `upperdir`/`workdir` is created
on that filesystem. Booting the default entry stays ephemeral.

With `--refresh`, an existing `hyprtk-persist` partition's exact geometry is read
before the write and re-created unchanged afterwards, so the filesystem is never
resized (and the ISO must not overlap it).

## Safety

The writer refuses: a partition instead of a whole disk, a disk with mounted
partitions, the disk backing `/`, a target smaller than the ISO, an out-of-range
size, and (non-fatally, as a warning) an ISO that does not look like hyprtk.
Non-interactive runs require confirmation unless `-y`.

## Build

```bash
make test       # unittest suite (+ an sfdisk integration test)
make wheel      # sdist + wheel into python/dist (needs `pip install build`)
make zipapp     # single-file dist/hyprtk-usb.pyz (rich vendored in)
make install    # pip install --user .
```

Runtime dependencies: `rich`, and `sfdisk`/`mkfs.ext4`/`blkid`/`lsblk`/`findmnt`.

## Layout

```
python/hyprtk_usb/core.py   ISO + device discovery, planning, guardrails, writer
python/hyprtk_usb/ui.py     rich theme + hyprtk-styled panels and prompts
python/hyprtk_usb/cli.py    flags + TUI / non-interactive flows
python/tests/               unittest suite
.github/workflows/          release workflow (wheel + sdist + zipapp)
```
