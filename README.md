# Conversor de Video

A portable video converter for Windows and Mac with a native Spanish interface.
Uses [FFmpeg](https://ffmpeg.org/) to convert videos to H.264, H.265, or VP9.
Videos stay on your computer. No browser, uploads, or separate FFmpeg installation.

## Table of Contents

- [Usage](#usage)
- [Installation](#installation)
- [Example](#example)
- [Contributing](#contributing)
- [License](#license)

## Usage

1. Open **Conversor de Video**.
2. Select one video with **Seleccionar video**.
3. Choose **Códec** and **Formato**.
4. Click **Convertir**.

H.264 is the default codec.
The format defaults to the original format when the selected codec supports it; otherwise it defaults to MP4.

| Video codec | Output formats |
| --- | --- |
| H.264 | MP4, MOV, MKV |
| H.265 | MP4, MOV, MKV |
| VP9 | MP4, WebM, MKV |

The application writes `Compatible - <original name>.<format>` beside the original video.
If the name exists, it adds a number before the extension.
It never overwrites existing files or changes the original.

Progress, completion, and errors appear in the same window.
Closing during conversion asks for confirmation.
Stopping removes the unfinished output and preserves the original.

Conversions use the first video track and the first audio track, when present.
Additional tracks and subtitles are not copied.
HDR-to-SDR tone mapping is not supported.
System controls can follow the operating-system language; technical diagnostics keep their original text.

## Installation

Use the portable ZIP for your platform.
Get it from a GitHub release when available, or [build it from source](CONTRIBUTING.md#build).

| Platform | Portable ZIP | Application |
| --- | --- | --- |
| Windows 10 or later, Intel/AMD 64-bit | `Conversor de Video - Windows.zip` | `Conversor de Video.exe` |
| macOS 12 or later, Apple Silicon | `Conversor de Video - Mac.zip` | `Conversor de Video.app` |

1. Extract the ZIP.
2. Open the application.

Recipients do not need FFmpeg, Go, Python, or Node.js.
The ZIPs contain only the application.
License files remain in this repository; see [License](#license) before sharing an application.

Windows builds are unsigned. Mac builds have a local signature, not Apple notarization.
The operating system can require approval or block a downloaded application.
Do not disable security checks.

## Example

To convert an H.265 video for an editor that accepts H.264:

1. Select `Vacaciones.mp4`.
2. Choose **H.264** under **Códec** and **MP4** under **Formato**.
3. Click **Convertir**.

```text
Input:  Vacaciones.mp4                  (H.265)
Output: Compatible - Vacaciones.mp4     (H.264)
```

If the output already exists, the application creates `Compatible - Vacaciones 1.mp4`, then `Compatible - Vacaciones 2.mp4`.
The original video remains unchanged.

## Contributing

Open an issue or submit a pull request.
See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, builds, checks, and contribution guidelines.
GitHub provides [issue forms](.github/ISSUE_TEMPLATE/) and a [pull request template](.github/pull_request_template.md).
[GitHub Actions](.github/workflows/ci.yml) checks Go source, Windows compilation, and AppKit source.

Native Windows execution still needs confirmation on Windows.
Mac package checks exercise real native controls and file-selector cancellation.

### Contributors

Project credit: [Daload](https://github.com/daload).
Contributions from the community are welcome.

## License

The project's own source uses the [MIT License](LICENSE).
Copyright © Daload.

Bundled FFmpeg uses its separate [GPLv3-or-later license](assets/licenses/ffmpeg-gplv3.txt).
This repository contains the [FFmpeg attribution notice](assets/licenses/ffmpeg-notice.txt), official project links, and license text.
The applications and their ZIPs do not include separate license or attribution files.
The MIT License does not replace FFmpeg's license or distribution requirements.
Distributors must provide recipients with the applicable licenses separately.
Attribution alone does not fulfill the [GPL corresponding-source requirements](https://www.gnu.org/licenses/gpl-3.0.html#section6), even for free distribution.
The packages do not include verified matching corresponding source.
