# Changelog

All notable changes to **hyprtk-usb** are documented in this file.
Dates are in `YYYY-MM-DD` format.

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
