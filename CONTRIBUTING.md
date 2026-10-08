# Contributing

## Submit a contribution

Use the GitHub issue forms to report a bug or propose a feature.
Include your operating system, codec, format, reproduction steps, and expected result.
Remove personal file paths and private information from diagnostics.
Do not attach personal videos.

Keep pull requests focused on one change.
Use the pull request template to explain the problem, the change, and the checks you ran.
Keep the native interface in Spanish.

## Build

Install Go and Python 3.11 or later.
Mac builds also require Apple's Command Line Tools and CGO.
Portable builds use the Go toolchain declared in [go.mod](go.mod) to retain macOS 12 support.
Go downloads that toolchain on first use if it is missing.

Place FFmpeg at the path for each target:

```text
.deps/windows/ffmpeg.exe
.deps/macos/ffmpeg
```

Use [Gyan's Windows essentials build](https://www.gyan.dev/ffmpeg/builds/) for Windows.
Use an [Apple Silicon build from Martin Riedl](https://ffmpeg.martin-riedl.de/) for Mac.
Keep the generated icons in `assets/`.

Run the command for your target from the project folder:

```sh
python3 scripts/build.py windows
python3 scripts/build.py macos
```

On Mac, `python3 scripts/build.py all` builds both platforms.
Mac application packaging requires macOS.
The Windows build downloads the pinned `github.com/akavel/rsrc@v0.10.2` resource compiler on first use.

Builds write applications to `dist/windows/` and `dist/macos/`, and shareable ZIPs to `dist/`.
Keep [LICENSE](LICENSE), the [FFmpeg license](assets/licenses/ffmpeg-gplv3.txt), and the [FFmpeg notice](assets/licenses/ffmpeg-notice.txt) intact.
These files remain in the repository; builds do not include them in applications or ZIPs.

## Run from source

Install Go and a native FFmpeg binary.
From the project folder, run:

```sh
go run ./cmd/convertidor --ffmpeg .deps/macos/ffmpeg
```

On Windows, use `.deps/windows/ffmpeg.exe` instead.
An optional video path preselects one video.

The application uses AppKit on Mac and Win32 controls on Windows.
Mac builds require Apple's Command Line Tools and CGO.
The conversion controller has no operating-system dependencies.
Linux runs controller and conversion tests, but does not provide an application window.

## Check source changes

Run Go checks from the project folder:

```sh
go test ./...
go vet ./...
```

To include real codec/container conversion checks, install FFmpeg and ffprobe.
On a Mac with the local binary, run:

```sh
FFMPEG_TEST_BINARY="$PWD/.deps/macos/ffmpeg" go test ./...
```

The controller tests cover defaults, conversion state, invalid choices, progress, errors, and shutdown.

## Check portable packages

Build with [scripts/build.py](scripts/build.py).
Then run:

```sh
python3 assets/test_icons.py
python3 scripts/test_packages.py
```

Package checks require both generated applications and a Mac.
They open native Mac windows, cancel a native file selector, and convert generated test videos.
They also check error cleanup, relocated applications, bundled FFmpeg, icons, and signatures.
They do not convert user videos.

The developer-only `--test-ui <video>` flag exercises native selectors and the **Convertir** action.
It writes an H.264 MP4 beside the supplied video, prints the final state, and exits.
Use only a test video with this flag.
Check native Windows behavior on Windows before distribution.

## Publish portable packages

Build and check both applications before publishing a release.
Upload the generated platform ZIPs as GitHub release assets, rather than committing binaries to the source repository.
Include the supported platforms and any unverified platform behavior in the release notes.
Before distribution, resolve the [FFmpeg source requirements](README.md#license).
Provide recipients with the applicable licenses separately; the generated ZIPs contain only the application.
License notices and official project links are not a complete corresponding-source package.

## Project layout

| Path | Purpose |
| --- | --- |
| `cmd/convertidor/` | Application entry point |
| `internal/` | Conversion, native controls, platform helpers, and bundled FFmpeg |
| `assets/` | Editable icon, platform icons, and FFmpeg license |
| `scripts/` | Builds and native package checks |
| `.deps/` | Local FFmpeg binaries; ignored by Git |
| `dist/` | Generated applications and ZIPs; ignored by Git |

The codec/format contract lives in [options.go](internal/converter/options.go).
Shared native control state lives in [controller.go](internal/desktop/controller.go).
Keep the [editable SVG](assets/convertidor-icon.svg).
Run `swift assets/render-icon.swift` on Mac to regenerate the PNG, ICO, and ICNS.
Rebuild the applications after changing the artwork.

## Keep the repository small

Do not commit `.deps/` or `dist/`.
Keep conversion choices in [options.go](internal/converter/options.go), not duplicate frontend lists.
Update native controls only on their operating-system event thread.
Keep filenames and errors as text in the window.
Keep all application-owned labels, messages, dialogs, and errors in Spanish.
Keep FFmpeg's original technical diagnostics and third-party licences unchanged.
Do not add a browser runtime, an HTTP server, or external services.
Do not overwrite existing videos.
