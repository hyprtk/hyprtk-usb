import os
import shutil
import subprocess
import tempfile
import unittest

from hyprtk_usb import core


@unittest.skipUnless(shutil.which("sfdisk"), "sfdisk not installed")
class AppendPartitionIntegration(unittest.TestCase):
    """Exercise the real sfdisk against a file with an isohybrid-like MBR."""

    def test_append_preserves_entries(self):
        d = tempfile.mkdtemp()
        target = os.path.join(d, "stick.img")
        with open(target, "wb") as f:
            f.truncate(64 << 20)

        r = core.ExecRunner()
        craft = "label: dos\nunit: sectors\n\nstart=64, size=20000, type=0, bootable\nstart=20064, size=10000, type=ef\n"
        r.run("sfdisk", target, stdin=craft)

        total = (64 << 20) // 512
        start = 32768
        core.append_partition(r, target, start, total - start)

        dump = subprocess.run(["sfdisk", "-d", target], capture_output=True, check=True).stdout.decode()
        norm = "".join(dump.split())
        for want in ("start=64,size=20000,type=0,bootable", "start=20064,size=10000,type=ef",
                     "start=32768", "type=83"):
            self.assertIn(want, norm)


if __name__ == "__main__":
    unittest.main()
