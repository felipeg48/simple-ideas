#!/bin/sh
# Builds "si Studio.app" (universal: Apple silicon + Intel) into desktop/dist and
# zips it. The app holds the si binary and the native shell. It is signed with
# the self-signed identity "simple-ideas Self-Signed" when that is in the
# keychain (make desktop-cert creates it once), else ad-hoc; no Developer ID,
# no notarisation: see docs/desktop.md.
set -eu
cd "$(dirname "$0")"
VERSION=$(sed -n 's/^const version = "\(.*\)"$/\1/p' ../cmd/si/main.go)
APP="dist/si Studio.app"
rm -rf dist
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
export MACOSX_DEPLOYMENT_TARGET=13.0
for arch in arm64 amd64; do
	carch=$arch; [ "$arch" = amd64 ] && carch=x86_64
	(cd .. && CGO_ENABLED=0 GOOS=darwin GOARCH=$arch go build -trimpath -ldflags "-s -w" -o "desktop/dist/si-$arch" ./cmd/si)
	CGO_ENABLED=1 GOOS=darwin GOARCH=$arch CC="clang -arch $carch" go build -trimpath -ldflags "-s -w" -o "dist/shell-$arch" .
done
lipo -create dist/si-arm64 dist/si-amd64 -output "$APP/Contents/MacOS/si"
lipo -create dist/shell-arm64 dist/shell-amd64 -output "$APP/Contents/MacOS/si-studio"
rm dist/si-arm64 dist/si-amd64 dist/shell-arm64 dist/shell-amd64
sed "s/__VERSION__/$VERSION/g" Info.plist > "$APP/Contents/Info.plist"
cp AppIcon.icns "$APP/Contents/Resources/"
IDENTITY=${SI_SIGN_IDENTITY:-simple-ideas Self-Signed}
if security find-certificate -c "$IDENTITY" >/dev/null 2>&1; then
	codesign --force --deep --sign "$IDENTITY" "$APP" # a stable identity: macOS keeps the app's permissions across updates
else
	codesign --force --deep --sign - "$APP" # ad-hoc: Apple silicon will not run unsigned code
fi
(cd dist && ditto -c -k --keepParent "si Studio.app" "si-Studio-$VERSION-macos.zip")
echo "built desktop/dist/si Studio.app and desktop/dist/si-Studio-$VERSION-macos.zip"
