# Changelog

All notable changes to **hyprtk-usb** are documented in this file.
Dates are in `YYYY-MM-DD` format.

## [Unreleased]

### Added

- **GTK 3 GUI** (`hyprtk-usb-gui`) over the same `core` backend: an ISO picker
  (auto-list + Browse), a device picker, options, a review page and a live
  progress bar. It runs **unprivileged** — the write is performed by
  `hyprtk_usb.helper` (`hyprtk-usb-helper`) launched through `pkexec`, so the GTK
  app never runs as root and pkexec's stripped environment can't break the display.
- A `.desktop` entry + icon under `python/data/`, installed by `make install`.
- `core.resolve_target` (shared by the CLI, GUI and helper) and the
  `hyprtk-usb-helper` / `hyprtk-usb-gui` console scripts.
- A helper end-to-end test (runs the helper as a subprocess against file targets).

## [0.2.0] - 2026-09-22

### Added

- **Python implementation** (`python/`) — a `rich` UI with the hyprtk theme:
  double-border panels, the mauve/cyan palette resolved from pywal, and inline
  prompts (no full-screen takeover). Same flags and behaviour as the Go binary.
  - `hyprtk_usb/core.py` — ISO/device discovery, plan geometry, guardrails, writer.
  - `hyprtk_usb/ui.py` — the themed panels, prompts and progress bar.
  - `hyprtk_usb/cli.py` — flags + TUI / non-interactive flows.
  - `tests/` — unittest suite plus an `sfdisk` integration test.

### Removed

- The Go implementation (`cmd/`, `internal/`, `go.mod`) — the Python version is
  now the only implementation.

### Changed

- The release workflow now builds a **wheel**, an **sdist** and a single-file
  **zipapp** (`hyprtk-usb.pyz`, `rich` vendored in) instead of a static Go binary.

## [0.1.0] - 2026-09-22

### Added

- Initial Go implementation: `internal/usb` (ISO + device discovery, planning,
  guardrails, writer with unit and `sfdisk` integration tests) and
  `cmd/hyprtk-usb` (interactive Bubble Tea TUI + non-interactive CLI with the
  shell script's flags). Behaviour matched the retired `hyprtk-usb` shell script:
  stream the iso-hybrid MBR ISO, append a 1 MiB-aligned `hyprtk-persist` ext4
  partition with `sfdisk --append` (existing entries preserved), and honour the
  ISO's `cow_label=hyprtk-persist` contract. `--refresh` preserved an existing
  partition's geometry; `--test`/`HYPRTK_USB_TEST=1` allowed a file target.
