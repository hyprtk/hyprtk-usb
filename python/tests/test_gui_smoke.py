"""Headless smoke test for the Qt GUI.

Runs the wizard's pure UI steps under the offscreen Qt platform. No hardware,
no root: it only proves the Qt widgets construct and the palette/QSS build. The
logic itself is covered by test_core / test_helper.
"""

from __future__ import annotations

import os
import unittest

os.environ.setdefault("QT_QPA_PLATFORM", "offscreen")
os.environ.setdefault("HYPRTK_USB_TEST", "1")

try:
    from PySide6.QtWidgets import QApplication

    from hyprtk_usb import gui
    from hyprtk_usb.palette import Palette

    HAVE_QT = True
except Exception:  # pragma: no cover - exercised only without PySide6
    HAVE_QT = False


@unittest.skipUnless(HAVE_QT, "PySide6 not available")
class GuiSmokeTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.app = QApplication.instance() or QApplication([])

    def test_qss_builds_from_palette(self) -> None:
        qss = gui.build_qss(Palette())
        self.assertIn("QFrame#panel", qss)
        self.assertIn("QProgressBar", qss)

    def test_pure_steps_construct(self) -> None:
        win = gui.Window()
        # Only the steps that do not touch hardware.
        for step in ("iso", "options", "progress", "done", "error"):
            win._error = "boom"
            win.step = step
            try:
                win.show_step()
            except Exception as exc:  # noqa: BLE001
                self.fail(f"step {step!r} failed to construct: {exc}")
        win.close()

    def test_switch_toggles(self) -> None:
        sw = gui.Switch(False)
        self.assertFalse(sw.isChecked())
        sw.setChecked(True)
        self.assertTrue(sw.isChecked())


if __name__ == "__main__":
    unittest.main()
