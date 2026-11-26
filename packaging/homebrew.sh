#!/bin/sh
# Writes the Homebrew tap's files for one release: Formula/si.rb (the si
# binaries of the release) and Casks/si-studio.rb (the Mac app).
#   packaging/homebrew.sh 4.5.0 dist/SHA256SUMS.txt <sha256 of the app zip> <tap dir>
# release.yml runs it and pushes the tap (felipeg48/homebrew-tap).
set -eu
v=$1 sums=$2 appsha=$3 out=$4
sum() { awk -v f="si_v${v}_$1" '$2 == f { print $1 }' "$sums"; }
for p in darwin_arm64 darwin_amd64 linux_arm64 linux_amd64; do
	[ -n "$(sum $p)" ] || { echo "homebrew.sh: no si_v${v}_$p in $sums" >&2; exit 1; }
done
base="https://github.com/felipeg48/simple-ideas/releases/download/v$v"
mkdir -p "$out/Formula" "$out/Casks"
cat > "$out/Formula/si.rb" <<EOF
# Written by simple-ideas' packaging/homebrew.sh on every release; do not edit.
class Si < Formula
  desc "Markdown into animated, fluid HTML slide decks, articles, guides and mind maps"
  homepage "https://github.com/felipeg48/simple-ideas"
  version "$v"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "$base/si_v${v}_darwin_arm64"
      sha256 "$(sum darwin_arm64)"
    end
    on_intel do
      url "$base/si_v${v}_darwin_amd64"
      sha256 "$(sum darwin_amd64)"
    end
  end
  on_linux do
    on_arm do
      url "$base/si_v${v}_linux_arm64"
      sha256 "$(sum linux_arm64)"
    end
    on_intel do
      url "$base/si_v${v}_linux_amd64"
      sha256 "$(sum linux_amd64)"
    end
  end

  def install
    bin.install Dir["si_v*"].first => "si"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/si --version")
  end
end
EOF
cat > "$out/Casks/si-studio.rb" <<EOF
# Written by simple-ideas' packaging/homebrew.sh on every release; do not edit.
cask "si-studio" do
  version "$v"
  sha256 "$appsha"

  url "https://github.com/felipeg48/simple-ideas/releases/download/v#{version}/si-Studio-#{version}-macos.zip"
  name "si Studio"
  desc "Editor for si decks, articles, guides and mind maps"
  homepage "https://github.com/felipeg48/simple-ideas"

  depends_on macos: ">= :ventura"

  app "si Studio.app"

  caveats <<~EOS
    si Studio is self-signed, not notarised by Apple yet. If macOS says it
    cannot be opened, run once:
      xattr -dr com.apple.quarantine "/Applications/si Studio.app"
  EOS
end
EOF
