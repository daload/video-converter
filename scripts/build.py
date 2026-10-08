import argparse
import os
from pathlib import Path
import plistlib
import shutil
import subprocess
import sys
import tempfile
import zipfile


PROJECT = Path(__file__).resolve().parent.parent
DIST = PROJECT / "dist"
APP_NAME = "Conversor de Video"


def required(path):
    if not path.is_file():
        raise FileNotFoundError(f"Falta un archivo necesario: {path}. Consulta README.md para configurar el proyecto.")
    return path


def build_go(work, platform, destination):
    source = work / platform
    source.mkdir()
    shutil.copy2(PROJECT / "go.mod", source / "go.mod")
    for folder in ("cmd", "internal"):
        shutil.copytree(PROJECT / folder, source / folder)
    toolchain = next((line.split()[1] for line in (PROJECT / "go.mod").read_text().splitlines()
                      if line.startswith("toolchain ")), None)
    if toolchain is None:
        raise RuntimeError("Declara la versión de Go para las aplicaciones portátiles en go.mod.")
    flags = "-s -w"
    if platform == "windows":
        subprocess.run(
            ["go", "run", "github.com/akavel/rsrc@v0.10.2", "-arch", "amd64",
             "-ico", str(required(PROJECT / "assets/convertidor-icon.ico")),
             "-o", str(source / "cmd/convertidor/icon_windows_amd64.syso")],
            cwd=PROJECT, env=dict(os.environ, GOTOOLCHAIN=toolchain), check=True,
        )
        flags += " -H=windowsgui"
    env = dict(os.environ, GOOS="windows" if platform == "windows" else "darwin",
               GOARCH="amd64" if platform == "windows" else "arm64",
               CGO_ENABLED="0" if platform == "windows" else "1",
               MACOSX_DEPLOYMENT_TARGET="12.0", GOTOOLCHAIN=toolchain)
    subprocess.run(
        ["go", "build", "-trimpath", f"-ldflags={flags}", "-o", str(destination), "./cmd/convertidor"],
        cwd=source, env=env, check=True,
    )


def replace_file(source, destination):
    destination.parent.mkdir(parents=True, exist_ok=True)
    descriptor, name = tempfile.mkstemp(prefix=".convertidor-", dir=destination.parent)
    os.close(descriptor)
    temporary = Path(name)
    try:
        shutil.copyfile(source, temporary)
        temporary.chmod(0o644)
        temporary.replace(destination)
    finally:
        temporary.unlink(missing_ok=True)


def windows(work):
    executable = work / f"{APP_NAME}.exe"
    ffmpeg = required(PROJECT / ".deps/windows/ffmpeg.exe")
    build_go(work, "windows", executable)
    with zipfile.ZipFile(executable, "a", zipfile.ZIP_DEFLATED, compresslevel=6) as archive:
        archive.write(ffmpeg, "ffmpeg.exe")
    output = DIST / f"windows/{APP_NAME}.exe"
    replace_file(executable, output)
    archive = work / f"{APP_NAME} - Windows.zip"
    with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED) as package:
        package.write(executable, f"{APP_NAME}.exe")
    replace_file(archive, DIST / archive.name)
    print(f"Creado: {output}")


def macos(work):
    if sys.platform != "darwin":
        raise RuntimeError("Crea la aplicación de Mac en un Mac.")
    ffmpeg = required(PROJECT / ".deps/macos/ffmpeg")
    app = work / f"{APP_NAME}.app"
    resources = app / "Contents/Resources"
    resources.mkdir(parents=True)
    executable = app / f"Contents/MacOS/{APP_NAME}"
    executable.parent.mkdir()
    build_go(work, "macos", executable)
    for source, name in [(ffmpeg, "ffmpeg"),
                         (required(PROJECT / "assets/convertidor-icon.icns"), "Convertidor.icns")]:
        shutil.copy2(source, resources / name)
    (resources / "ffmpeg").chmod(0o755)
    info_path = app / "Contents/Info.plist"
    info = dict(CFBundleIdentifier="local.convertidor.mac", CFBundleName=APP_NAME,
                CFBundleDisplayName=APP_NAME, CFBundleIconFile="Convertidor.icns",
                CFBundleExecutable=APP_NAME, CFBundlePackageType="APPL",
                CFBundleDevelopmentRegion="es", CFBundleLocalizations=["es"],
                CFBundleInfoDictionaryVersion="6.0", NSPrincipalClass="NSApplication",
                NSHighResolutionCapable=True,
                CFBundleVersion="4", CFBundleShortVersionString="3.1",
                LSMinimumSystemVersion="12.0",
                CFBundleDocumentTypes=[{"CFBundleTypeName": "Archivo de video", "CFBundleTypeRole": "Viewer",
                                       "CFBundleTypeExtensions": ["*"], "LSItemContentTypes": ["public.data"], "LSHandlerRank": "None"}])
    with info_path.open("wb") as file:
        plistlib.dump(info, file)
    subprocess.run(["/usr/bin/codesign", "--force", "--sign", "-", str(executable)], check=True)
    subprocess.run(["/usr/bin/codesign", "--force", "--sign", "-", str(app)], check=True)
    archive = work / f"{APP_NAME} - Mac.zip"
    subprocess.run(["/usr/bin/ditto", "-c", "-k", "--keepParent", str(app), str(archive)], check=True)
    output = DIST / f"macos/{APP_NAME}.app"
    output.parent.mkdir(parents=True, exist_ok=True)
    if output.is_symlink():
        raise RuntimeError(f"No se puede reemplazar un enlace simbólico: {output}")
    if output.exists():
        shutil.rmtree(output)
    shutil.copytree(app, output)
    replace_file(archive, DIST / archive.name)
    print(f"Creado: {output}")


def main():
    parser = argparse.ArgumentParser(description="Crear las aplicaciones portátiles de Conversor de Video.")
    parser.add_argument("target", choices=["windows", "macos", "all"], nargs="?",
                        default="all" if sys.platform == "darwin" else "windows")
    target = parser.parse_args().target
    DIST.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".convertidor-build-", dir=DIST) as folder:
        work = Path(folder)
        if target in ("windows", "all"):
            windows(work)
        if target in ("macos", "all"):
            macos(work)


if __name__ == "__main__":
    main()
