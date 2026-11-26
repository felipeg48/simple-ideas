#!/bin/sh
# Writes the winget manifests for one release (simple-ideas.si, a portable
# si.exe) into <dir>/manifests/s/simple-ideas/si/<version>/, the layout of
# microsoft/winget-pkgs, where they go as a pull request.
#   packaging/winget.sh 4.5.0 dist/SHA256SUMS.txt <dir> [release date]
set -eu
v=$1 sums=$2 out=$3 day=${4:-$(date -u +%Y-%m-%d)}
sum() { awk -v f="si_v${v}_$1" '$2 == f { print toupper($1) }' "$sums"; }
for p in windows_amd64.exe windows_arm64.exe; do
	[ -n "$(sum $p)" ] || { echo "winget.sh: no si_v${v}_$p in $sums" >&2; exit 1; }
done
id=simple-ideas.si base="https://github.com/felipeg48/simple-ideas/releases/download/v$v"
d="$out/manifests/s/simple-ideas/si/$v"
mkdir -p "$d"
cat > "$d/$id.yaml" <<EOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.version.1.28.0.schema.json

PackageIdentifier: $id
PackageVersion: $v
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.28.0
EOF
cat > "$d/$id.installer.yaml" <<EOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.installer.1.28.0.schema.json

PackageIdentifier: $id
PackageVersion: $v
InstallerType: portable
Commands:
- si
ReleaseDate: $day
Installers:
- Architecture: x64
  InstallerUrl: $base/si_v${v}_windows_amd64.exe
  InstallerSha256: $(sum windows_amd64.exe)
- Architecture: arm64
  InstallerUrl: $base/si_v${v}_windows_arm64.exe
  InstallerSha256: $(sum windows_arm64.exe)
ManifestType: installer
ManifestVersion: 1.28.0
EOF
cat > "$d/$id.locale.en-US.yaml" <<EOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.defaultLocale.1.28.0.schema.json

PackageIdentifier: $id
PackageVersion: $v
PackageLocale: en-US
Publisher: simple-ideas
PublisherUrl: https://github.com/felipeg48
PublisherSupportUrl: https://github.com/felipeg48/simple-ideas/issues
Author: Felipe Gutierrez
PackageName: si
PackageUrl: https://github.com/felipeg48/simple-ideas
License: Apache-2.0
LicenseUrl: https://github.com/felipeg48/simple-ideas/blob/main/LICENSE
ShortDescription: Markdown into animated, fluid HTML slide decks, articles, guides and mind maps.
Description: si turns Markdown into animated, fluid HTML slide decks, articles, guides, books and mind maps. si play presents with live reload and a teleprompter; si build writes one self-contained HTML file.
Moniker: si
Tags:
- markdown
- slides
- presentation
- teleprompter
- static-site
ReleaseNotesUrl: https://github.com/felipeg48/simple-ideas/releases/tag/v$v
ManifestType: defaultLocale
ManifestVersion: 1.28.0
EOF
