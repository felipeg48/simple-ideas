---
title: Install
description: Install si with Homebrew, a .deb or .rpm package, or one file for macOS, Linux and Windows; si Studio for the Mac.
---

# Install

The latest si is **v$SI_VERSION**:
[release notes](https://github.com/felipeg48/simple-ideas/releases/tag/v$SI_VERSION).
si is one program with nothing else to install; Chrome and ffmpeg are only
needed for PDF, picture and video exports.

## macOS and Linux: Homebrew

```bash
brew install felipeg48/tap/si
si --version
```

`brew upgrade si` brings the next release.

## Debian, Ubuntu: a .deb

```bash
curl -LO https://github.com/felipeg48/simple-ideas/releases/download/v$SI_VERSION/simple-ideas_$SI_VERSION-1_amd64.deb
sudo apt install ./simple-ideas_$SI_VERSION-1_amd64.deb
```

On ARM (a Raspberry Pi, a Graviton server), take `_arm64.deb`. The package is
`simple-ideas`, the command `si`. Install the next release's `.deb` the same
way to upgrade; `sudo apt remove simple-ideas` takes it away.

## Fedora, RHEL, openSUSE: an .rpm

```bash
sudo dnf install https://github.com/felipeg48/simple-ideas/releases/download/v$SI_VERSION/simple-ideas-$SI_VERSION-1.x86_64.rpm
```

On ARM, take `.aarch64.rpm`; on openSUSE, `sudo zypper install` the same
address. `sudo dnf remove simple-ideas` takes it away.

## Windows

Download `si_v$SI_VERSION_windows_amd64.exe` (or `_arm64.exe`) from the
[latest release](https://github.com/felipeg48/simple-ideas/releases/latest),
rename it `si.exe` and put it in a folder on your `PATH`, then in a new
terminal:

```powershell
si --version
```

## One file, any system

Every release has the plain binaries, `si_v<version>_<os>_<arch>`, checked by
`SHA256SUMS.txt`:

```bash
chmod +x si_* && sudo mv si_* /usr/local/bin/si
si --version
```

Installed this way (or on Windows), si updates itself in place, checksum
verified:

```bash
si selfupdate
```

With Homebrew or a package, upgrade through it instead; `si selfupdate` says
so.

With Go 1.26 or newer:

```bash
go install github.com/felipeg48/simple-ideas/cmd/si@latest
```

## si Studio for the Mac

```bash
brew install --cask felipeg48/tap/si-studio
```

Or take `si-Studio-<version>-macos.zip` from the release, unzip it and move
**si Studio.app** to Applications. It runs the Studio for any folder you open
and tells you when a newer release is out. It is not notarised by Apple yet:
if macOS says it cannot be opened, run once
`xattr -dr com.apple.quarantine "/Applications/si Studio.app"`.

## First steps

```bash
si new talk          # a starter deck
si play talk.md --open
si --web .           # the Studio in the browser
```
