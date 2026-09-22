import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

from hyprtk_usb import core

PY_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


@unittest.skipUnless(shutil.which("sfdisk"), "sfdisk not installed")
class HelperEndToEnd(unittest.TestCase):
    """Run the privileged helper as a subprocess against file targets."""

    def test_write_emits_progress_and_partitions(self):
        d = tempfile.mkdtemp()
        iso = os.path.join(d, "fake.iso")
        target = os.path.join(d, "stick.img")
        with open(iso, "wb") as f:
            f.truncate(16 << 20)
        with open(target, "wb") as f:
            f.truncate(64 << 20)

        # The ISO carries its own isohybrid MBR; the copy writes it to the target,
        # then the helper appends the persistence partition (the real flow).
        r = core.ExecRunner()
        r.run(
            "sfdisk", iso,
            stdin="label: dos\nunit: sectors\n\nstart=64, size=20000, type=0, bootable\n"
                  "start=20064, size=10000, type=ef\n",
        )

        env = dict(os.environ, PYTHONPATH=PY_DIR)
        proc = subprocess.run(
            [sys.executable, "-m", "hyprtk_usb.helper",
             "--iso", iso, "--target", target, "--size", "rest", "--test"],
            capture_output=True, text=True, env=env,
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)

        stages = [json.loads(ln)["stage"] for ln in proc.stdout.splitlines() if ln.strip()]
        self.assertEqual(stages[0], "plan")
        self.assertIn("copy", stages)
        self.assertIn("partition", stages)
        self.assertEqual(stages[-1], "done")

        dump = subprocess.run(["sfdisk", "-d", target], capture_output=True, text=True).stdout
        norm = "".join(dump.split())
        self.assertIn("start=64,size=20000,type=0,bootable", norm)  # preserved
        self.assertIn("type=83", norm)                              # appended


if __name__ == "__main__":
    unittest.main()
