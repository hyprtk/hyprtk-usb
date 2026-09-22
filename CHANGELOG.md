# Changelog

All notable changes to **hyprtk-usb** are documented in this file.
Dates are in `YYYY-MM-DD` format.

## [Unreleased] - 2026-09-22

### Added

- Initial Go rewrite of the `hyprtk-usb` shell script.
  - `internal/usb`: ISO + device discovery, plan/geometry, guardrails and the
    writer, with unit tests and an `sfdisk` integration test.
  - `cmd/hyprtk-usb`: interactive Bubble Tea TUI (device picker, options, typed
    confirmation, live `dd` progress) and a non-interactive CLI with the same
    flags as the script (`--iso`, `--target`, `--size`, `--no-persist`,
    `--refresh`, `--dry-run`, `-y`).
  - Behaviour matches the script: `dd` the iso-hybrid MBR ISO, append a
    1 MiB-aligned `hyprtk-persist` ext4 partition with `sfdisk --append`
    (existing entries preserved), keep the ISO's persistence contract
    (`cow_label=hyprtk-persist`).
  - `--refresh` preserves an existing persistence partition's exact geometry.
  - A `--test` / `HYPRTK_USB_TEST=1` seam allows a regular-file target for
    smoke testing without hardware.

- A peer **Python** implementation in `python/` (`rich` UI, hyprtk theme:
  double-border panels, mauve/cyan palette, inline prompts). Same flags and
  behaviour as the Go binary; `python -m unittest discover -s python/tests`.
