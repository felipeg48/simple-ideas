---
title: Download
description: The si binary for macOS, Linux and Windows, and si Studio for the Mac.
---

# Download

## The si binary

Pick your system on the
[latest release](https://github.com/felipeg48/simple-ideas/releases/latest):
one file, `si_<version>_<os>_<arch>`, checked by `SHA256SUMS.txt`.

```bash
# macOS / Linux
chmod +x si_* && sudo mv si_* /usr/local/bin/si
si --version
```

On Windows, put `si_<version>_windows_amd64.exe` somewhere on your `PATH` as
`si.exe`.

With Go 1.26 or newer:

```bash
go install github.com/felipeg48/simple-ideas/cmd/si@latest
```

An installed `si` updates itself in place:

```bash
si selfupdate
```

## si Studio for the Mac

The same release has `si-Studio-<version>-macos.zip`: unzip it and move
**si Studio.app** to Applications. It runs the Studio for any folder you open
and tells you when a newer release is out.

## First steps

```bash
si new talk          # a starter deck
si play talk.md --open
si --web .           # the Studio in the browser
```
