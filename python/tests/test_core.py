import os
import tempfile
import unittest

from hyprtk_usb import core


class FakeRunner(core.Runner):
    def __init__(self, outputs=None):
        self.outputs = outputs or {}
        self.calls = []

    def output(self, *args):
        return self.outputs.get(args[0], b"")

    def run(self, *args, stdin=""):
        self.calls.append((args, stdin))

    def called(self, needle):
        return any(needle in " ".join(a) for a, _ in self.calls)


def iso_of(sectors):
    return core.ISO(path="x.iso", size=sectors * 512)


class ParseSize(unittest.TestCase):
    def test_forms(self):
        avail = 1_000_000
        self.assertEqual(core.parse_size("", avail), avail)
        self.assertEqual(core.parse_size("rest", avail), avail)
        self.assertEqual(core.parse_size("50%", avail), avail * 50 // 100)
        self.assertEqual(core.parse_size("1G", avail), 1024**3 // 512)
        self.assertEqual(core.parse_size("512M", avail), 512 * 1024**2 // 512)
        self.assertEqual(core.parse_size("12345", avail), 12345)

    def test_bad(self):
        for spec in ("0%", "101%", "1T", "abc", "-5"):
            with self.assertRaises(core.UsbError):
                core.parse_size(spec, 1000)


class Helpers(unittest.TestCase):
    def test_align_up(self):
        self.assertEqual(core.align_up(2049, 2048), 4096)
        self.assertEqual(core.align_up(2048, 2048), 2048)

    def test_partition_path(self):
        self.assertEqual(core.partition_path("/dev/sda", 3), "/dev/sda3")
        self.assertEqual(core.partition_path("/dev/nvme0n1", 3), "/dev/nvme0n1p3")
        self.assertEqual(core.partition_path("/dev/loop0", 3), "/dev/loop0p3")
        self.assertEqual(core.partition_path("/dev/mmcblk0", 3), "/dev/mmcblk0p3")

    def test_next_free_part_num(self):
        dev = core.Device(path="/dev/sda", name="sda", size=0,
                          parts=[core.Partition("", 0, 0, "", "", 1), core.Partition("", 0, 0, "", "", 2)])
        self.assertEqual(core.next_free_part_num(dev), 3)
        full = core.Device(path="/dev/sda", name="sda", size=0,
                           parts=[core.Partition("", 0, 0, "", "", n) for n in (1, 2, 3, 4)])
        with self.assertRaises(core.UsbError):
            core.next_free_part_num(full)


class BuildPlan(unittest.TestCase):
    def test_no_persist(self):
        plan = core.build_plan(iso_of(1000), core.Device("/dev/sda", "sda", 10_000 * 512), core.Options(persist=False))
        self.assertEqual(plan.mode, core.MODE_NONE)

    def test_fresh(self):
        dev = core.Device(path="/dev/sda", name="sda", size=100_000 * 512,
                          parts=[core.Partition("", 0, 0, "", "", 1), core.Partition("", 0, 0, "", "", 2)])
        plan = core.build_plan(iso_of(1000), dev, core.Options(persist=True, size="rest"))
        self.assertEqual(plan.mode, core.MODE_FRESH)
        self.assertEqual(plan.start_sectors, 2048)
        self.assertEqual(plan.size_sectors, (100_000 - 2048) // 2048 * 2048)
        self.assertEqual(plan.partition_num, 3)
        self.assertEqual(plan.partition_dev, "/dev/sda3")

    def test_refresh(self):
        dev = core.Device(path="/dev/sda", name="sda", size=100_000 * 512, parts=[
            core.Partition("", 0, 0, "", "", 1),
            core.Partition("", 0, 0, "", "", 2),
            core.Partition("/dev/sda3", 50_000 * 512, 4096 * 512, core.PERSIST_LABEL, "", 3),
        ])
        plan = core.build_plan(iso_of(1000), dev, core.Options(persist=True, refresh=True))
        self.assertEqual(plan.mode, core.MODE_REFRESH)
        self.assertEqual((plan.start_sectors, plan.size_sectors), (4096, 50_000))

    def test_refresh_overlap(self):
        dev = core.Device(path="/dev/sda", name="sda", size=200_000 * 512, parts=[
            core.Partition("/dev/sda3", 50_000 * 512, 4096 * 512, core.PERSIST_LABEL, "", 3),
        ])
        with self.assertRaises(core.UsbError):
            core.build_plan(iso_of(100_000), dev, core.Options(persist=True, refresh=True))

    def test_too_small(self):
        with self.assertRaises(core.UsbError):
            core.build_plan(iso_of(100_000), core.Device("/dev/sda", "sda", 100_500 * 512), core.Options(persist=True))


class ValidateTarget(unittest.TestCase):
    def base(self):
        return dict(iso_size=100, root="/dev/nvme0n1")

    def test_ok(self):
        core.validate_target(core.Device("/dev/sda", "sda", 1000), **self.base())

    def test_rejections(self):
        with self.assertRaises(core.UsbError):
            core.validate_target(core.Device("", "", 1000), **self.base())
        with self.assertRaises(core.UsbError):
            core.validate_target(core.Device("/dev/nvme0n1", "x", 1000), **self.base())
        with self.assertRaises(core.UsbError):
            core.validate_target(core.Device("/dev/sda", "sda", 50), **self.base())
        mounted = core.Device("/dev/sda", "sda", 1000,
                             parts=[core.Partition("", 0, 0, "", "/run/media/x/EFI", 1)])
        with self.assertRaises(core.UsbError):
            core.validate_target(mounted, **self.base())

    def test_root_allowed_in_test_mode(self):
        core.validate_target(core.Device("/dev/nvme0n1", "x", 1000), iso_size=100, root="/dev/nvme0n1", test_mode=True)


class ListDevices(unittest.TestCase):
    SAMPLE = b"""{"blockdevices":[
      {"name":"sda","path":"/dev/sda","size":8589934592,"type":"disk","rm":true,"tran":"usb","model":"Ultra Fit",
       "mountpoint":null,"partn":null,"label":null,"start":null,
       "children":[
         {"name":"sda1","path":"/dev/sda1","size":2345678,"type":"part","partn":1,"mountpoint":null,"label":"HYPRTK_202609","start":32768},
         {"name":"sda2","path":"/dev/sda2","size":273684480,"type":"part","partn":2,"mountpoint":"/run/media/x/EFI","label":"EFI","start":2376496}
       ]},
      {"name":"loop0","path":"/dev/loop0","size":67108864,"type":"loop","rm":false,"tran":null,"model":null,
       "mountpoint":null,"partn":null,"label":null,"start":null,"children":[]}
    ]}"""

    def test_parse(self):
        r = FakeRunner({"lsblk": self.SAMPLE})
        devs = core.list_devices(r)
        self.assertEqual(len(devs), 2)
        sda = devs[0]
        self.assertEqual(sda.path, "/dev/sda")
        self.assertTrue(sda.removable)
        self.assertEqual(sda.mounts(), ["/run/media/x/EFI"])
        self.assertEqual(sda.persist_partition(), None)

    def test_root_disk(self):
        r = FakeRunner({"findmnt": b"/dev/nvme0n1p2\n", "lsblk": b"nvme0n1\n"})
        self.assertEqual(core.root_disk(r), "/dev/nvme0n1")


class Write(unittest.TestCase):
    def _files(self):
        d = tempfile.mkdtemp()
        iso_path = os.path.join(d, "hyprtk.iso")
        target = os.path.join(d, "target")
        with open(iso_path, "wb") as f:
            f.write(b"HYPRTK-ISO-CONTENTS")
        open(target, "wb").close()
        return iso_path, target

    def test_fresh_runs_sfdisk_and_mkfs(self):
        iso_path, target = self._files()
        iso = core.open_iso(iso_path, FakeRunner())
        plan = core.Plan(iso=iso, target_path=target, target_size=0, mode=core.MODE_FRESH,
                         start_sectors=2048, size_sectors=1000, partition_dev=target + "3")
        r = FakeRunner()
        stages = []
        core.write(iso, plan, r, lambda p: stages.append(p.stage))
        self.assertTrue(r.called("sfdisk"))
        self.assertTrue(r.called("mkfs.ext4"))
        with open(target, "rb") as f:
            self.assertEqual(f.read(), b"HYPRTK-ISO-CONTENTS")
        for want in ("copy", "partition", "format", "done"):
            self.assertIn(want, stages)

    def test_no_persist_skips_partitioning(self):
        iso_path, target = self._files()
        iso = core.open_iso(iso_path, FakeRunner())
        plan = core.Plan(iso=iso, target_path=target, target_size=0, mode=core.MODE_NONE)
        r = FakeRunner()
        core.write(iso, plan, r)
        self.assertFalse(r.called("sfdisk"))
        self.assertFalse(r.called("mkfs.ext4"))

    def test_refresh_skips_mkfs(self):
        iso_path, target = self._files()
        iso = core.open_iso(iso_path, FakeRunner())
        plan = core.Plan(iso=iso, target_path=target, target_size=0, mode=core.MODE_REFRESH,
                         start_sectors=4096, size_sectors=8000, partition_dev=target + "3")
        r = FakeRunner()
        core.write(iso, plan, r)
        self.assertTrue(r.called("sfdisk"))
        self.assertFalse(r.called("mkfs.ext4"))


if __name__ == "__main__":
    unittest.main()
