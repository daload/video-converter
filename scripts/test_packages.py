import hashlib
import json
import os
from pathlib import Path
import plistlib
import shutil
import struct
import subprocess
import tempfile
import unittest
import zipfile


PROJECT = Path(__file__).resolve().parent.parent
DIST = PROJECT / "dist"
APP = DIST / "macos/Conversor de Video.app"
RESOURCES = APP / "Contents/Resources"
EXECUTABLE = APP / "Contents/MacOS/Conversor de Video"


def run(arguments):
    return subprocess.run(arguments, stdin=subprocess.DEVNULL, capture_output=True, text=True)


class PortablePackageTests(unittest.TestCase):
    def test_windows_gui_executable_contains_original_ffmpeg_and_flat_zip(self):
        executable = DIST / "windows/Conversor de Video.exe"
        with executable.open("rb") as file:
            self.assertEqual(file.read(2), b"MZ")
            file.seek(0x3C)
            pe = struct.unpack("<I", file.read(4))[0]
            file.seek(pe + 4)
            self.assertEqual(struct.unpack("<H", file.read(2))[0], 0x8664)
            file.seek(pe + 24 + 68)
            self.assertEqual(struct.unpack("<H", file.read(2))[0], 2, "Windows UI must not open a console")
        with zipfile.ZipFile(executable) as archive:
            self.assertEqual(archive.namelist(), ["ffmpeg.exe"])
            with archive.open("ffmpeg.exe") as bundled, (PROJECT / ".deps/windows/ffmpeg.exe").open("rb") as original:
                self.assertEqual(hashlib.file_digest(bundled, "sha256").digest(), hashlib.file_digest(original, "sha256").digest())
        with zipfile.ZipFile(DIST / "Conversor de Video - Windows.zip") as archive:
            self.assertEqual(archive.namelist(), ["Conversor de Video.exe"])
            self.assertIsNone(archive.testzip())

    def test_mac_bundle_and_shareable_zip_preserve_resources_and_signatures(self):
        with (APP / "Contents/Info.plist").open("rb") as file:
            info = plistlib.load(file)
        self.assertEqual(info["CFBundleIconFile"], "Convertidor.icns")
        self.assertEqual(info["LSMinimumSystemVersion"], "12.0")
        self.assertEqual(info["CFBundleExecutable"], "Conversor de Video")
        self.assertEqual(info["CFBundleName"], "Conversor de Video")
        self.assertEqual(info["CFBundleDisplayName"], "Conversor de Video")
        self.assertEqual(info["CFBundleDevelopmentRegion"], "es")
        self.assertEqual(info["CFBundleLocalizations"], ["es"])
        self.assertEqual(info["NSPrincipalClass"], "NSApplication")
        self.assertNotIn("NSAppleEventsUsageDescription", info)
        self.assertEqual((RESOURCES / "Convertidor.icns").read_bytes(), (PROJECT / "assets/convertidor-icon.icns").read_bytes())
        self.assertEqual({path.name for path in RESOURCES.iterdir()}, {"ffmpeg", "Convertidor.icns"})
        with (RESOURCES / "ffmpeg").open("rb") as bundled, (PROJECT / ".deps/macos/ffmpeg").open("rb") as original:
            self.assertEqual(hashlib.file_digest(bundled, "sha256").digest(), hashlib.file_digest(original, "sha256").digest())
        for target in (APP, EXECUTABLE, RESOURCES / "ffmpeg"):
            result = run(["/usr/bin/codesign", "--verify", "--strict", str(target)])
            self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse((RESOURCES / "Scripts").exists(), "The app must not use an AppleScript launcher")
        dependencies = run(["/usr/bin/otool", "-L", str(EXECUTABLE)])
        self.assertEqual(dependencies.returncode, 0, dependencies.stderr)
        self.assertIn("Cocoa.framework", dependencies.stdout)
        self.assertNotIn("WebKit", dependencies.stdout)
        metadata = run(["/usr/bin/otool", "-l", str(EXECUTABLE)])
        self.assertEqual(metadata.returncode, 0, metadata.stderr)
        self.assertIn("minos 12.0", metadata.stdout)
        toolchain = next(line.split()[1] for line in (PROJECT / "go.mod").read_text().splitlines()
                         if line.startswith("toolchain "))
        version = run(["go", "version", str(EXECUTABLE)])
        self.assertEqual(version.returncode, 0, version.stderr)
        self.assertIn(toolchain, version.stdout)
        with tempfile.TemporaryDirectory(prefix="convertidor-package-zip-") as folder:
            with zipfile.ZipFile(DIST / "Conversor de Video - Mac.zip") as archive:
                self.assertTrue(all(name.startswith("Conversor de Video.app/") or name.startswith("__MACOSX/")
                                    for name in archive.namelist()))
                self.assertFalse(any(Path(name).name in ("LICENSE.txt", "FFmpeg-LICENSE.txt",
                                                         "FFmpeg-NOTICE.txt", "FFmpeg-source.txt")
                                     for name in archive.namelist()))
                self.assertIsNone(archive.testzip())
            result = run(["/usr/bin/ditto", "-x", "-k", str(DIST / "Conversor de Video - Mac.zip"), folder])
            self.assertEqual(result.returncode, 0, result.stderr)
            extracted = Path(folder) / "Conversor de Video.app"
            result = run(["/usr/bin/codesign", "--verify", "--deep", "--strict", str(extracted)])
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue((extracted / "Contents/MacOS/Conversor de Video").stat().st_mode & 0o111)
            self.assertEqual({path.name for path in (extracted / "Contents/Resources").iterdir()},
                             {"ffmpeg", "Convertidor.icns"})

    def test_native_mac_controls_use_bundled_ffmpeg_and_number_collisions(self):
        with tempfile.TemporaryDirectory(prefix="convertidor-native-ui-test-") as folder:
            moved_app = Path(folder) / "Friend's copy ! &" / "Conversor de Video.app"
            shutil.copytree(APP, moved_app)
            resources = moved_app / "Contents/Resources"
            input_file = Path(folder) / 'Vídeo de vacaciones ! & "prueba".mov'
            result = run([str(resources / "ffmpeg"), "-v", "error", "-f", "lavfi", "-i", "testsrc2=size=64x64:rate=10",
                          "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100", "-t", "0.3", "-c:v", "libx264", "-c:a", "aac", str(input_file)])
            self.assertEqual(result.returncode, 0, result.stderr)
            original_hash = hashlib.sha256(input_file.read_bytes()).digest()
            existing = input_file.with_name(f"Compatible - {input_file.stem}.mp4")
            existing.write_bytes(b"keep the existing output")
            executable = moved_app / "Contents/MacOS/Conversor de Video"
            result = subprocess.run([str(executable), "--test-ui", str(input_file)],
                                    stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=45,
                                    env=dict(os.environ, PATH="/usr/bin:/bin"))
            self.assertEqual(result.returncode, 0, result.stderr)
            state = json.loads(result.stdout)
            self.assertEqual(state["name"], input_file.name)
            self.assertEqual(state["codec"], "h264")
            self.assertEqual(state["format"], "mp4")
            self.assertEqual(len(state["options"]), 3)
            self.assertEqual(state["status"], "completed", state["message"])
            self.assertTrue(state["message"].startswith("Guardado en: "), state["message"])
            self.assertEqual(state["options"][0]["label"], "H.264 - más compatible")
            expected = input_file.with_name(f"Compatible - {input_file.stem} 1.mp4")
            self.assertEqual(state["output"], str(expected))
            self.assertEqual(state["progress"], 100)
            self.assertEqual(existing.read_bytes(), b"keep the existing output")
            self.assertEqual(hashlib.sha256(input_file.read_bytes()).digest(), original_hash)
            result = run([str(resources / "ffmpeg"), "-hide_banner", "-i", str(expected), "-f", "null", "-"])
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("Video: h264", result.stderr)
            self.assertIn("Audio: aac", result.stderr)
            self.assertEqual(list(Path(folder).glob(".convertidor-*")), [])

    def test_native_mac_conversion_error_preserves_input_and_cleans_temporary_files(self):
        with tempfile.TemporaryDirectory(prefix="convertidor-native-error-") as folder:
            input_file = Path(folder) / "Not a video.mp4"
            input_file.write_bytes(b"not a video")
            result = subprocess.run([str(EXECUTABLE), "--test-ui", str(input_file)],
                                    stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=45)
            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertIn("No se pudo convertir el video.", result.stderr)
            self.assertEqual(input_file.read_bytes(), b"not a video")
            self.assertEqual(list(Path(folder).iterdir()), [input_file])


if __name__ == "__main__":
    unittest.main()
