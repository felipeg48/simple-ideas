#!/bin/sh
# Writes the Scoop bucket's manifest for one release: bucket/si.json (the
# Windows binaries of the release, saved as si.exe).
#   packaging/scoop.sh 4.5.0 dist/SHA256SUMS.txt <bucket dir>
# release.yml runs it and pushes the bucket (felipeg48/scoop-bucket).
set -eu
v=$1 sums=$2 out=$3
sum() { awk -v f="si_v${v}_$1" '$2 == f { print $1 }' "$sums"; }
for p in windows_amd64.exe windows_arm64.exe; do
	[ -n "$(sum $p)" ] || { echo "scoop.sh: no si_v${v}_$p in $sums" >&2; exit 1; }
done
base="https://github.com/felipeg48/simple-ideas/releases/download"
mkdir -p "$out/bucket"
cat > "$out/bucket/si.json" <<EOF
{
    "version": "$v",
    "description": "Markdown into animated, fluid HTML slide decks, articles, guides and mind maps",
    "homepage": "https://github.com/felipeg48/simple-ideas",
    "license": "Apache-2.0",
    "architecture": {
        "64bit": {
            "url": "$base/v$v/si_v${v}_windows_amd64.exe#/si.exe",
            "hash": "$(sum windows_amd64.exe)"
        },
        "arm64": {
            "url": "$base/v$v/si_v${v}_windows_arm64.exe#/si.exe",
            "hash": "$(sum windows_arm64.exe)"
        }
    },
    "bin": "si.exe",
    "checkver": "github",
    "autoupdate": {
        "architecture": {
            "64bit": {
                "url": "$base/v\$version/si_v\$version_windows_amd64.exe#/si.exe"
            },
            "arm64": {
                "url": "$base/v\$version/si_v\$version_windows_arm64.exe#/si.exe"
            }
        },
        "hash": {
            "url": "\$baseurl/SHA256SUMS.txt"
        }
    }
}
EOF
