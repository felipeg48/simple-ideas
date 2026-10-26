#!/usr/bin/env bash
# setup.sh — one-off setup for building si on macOS / Linux.
# Checks for Go (prints install instructions if missing), then builds ./bin/si.
set -euo pipefail

REQUIRED_GO="1.26"
cd "$(dirname "$0")"

say()  { printf '\033[1m%s\033[0m\n' "$*"; }
err()  { printf '\033[31m%s\033[0m\n' "$*" >&2; }

install_help() {
  err "Go is not installed (or not on your PATH)."
  echo
  echo "Install Go ${REQUIRED_GO} or newer, then re-run ./setup.sh :"
  case "$(uname -s)" in
    Darwin) echo "  • Homebrew:  brew install go"
            echo "  • or download the macOS package from https://go.dev/dl/" ;;
    Linux)  echo "  • Debian/Ubuntu:  sudo apt-get install golang-go   (check the version)"
            echo "  • Fedora:         sudo dnf install golang"
            echo "  • Arch:           sudo pacman -S go"
            echo "  • or the official tarball from https://go.dev/dl/ :"
            echo "      curl -LO https://go.dev/dl/go${REQUIRED_GO}.0.linux-amd64.tar.gz"
            echo "      sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go${REQUIRED_GO}.0.linux-amd64.tar.gz"
            echo "      export PATH=\$PATH:/usr/local/go/bin" ;;
    *)      echo "  • download Go from https://go.dev/dl/" ;;
  esac
}

if ! command -v go >/dev/null 2>&1; then
  install_help
  exit 1
fi

have="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
say "Found Go ${have}"
# warn (don't fail) if older than required — the build will tell you for sure
if [ -n "$have" ] && [ "$(printf '%s\n%s\n' "$REQUIRED_GO" "$have" | sort -V | head -1)" != "$REQUIRED_GO" ]; then
  err "Warning: go.mod needs Go ${REQUIRED_GO}+; you have ${have}. Upgrade if the build fails."
fi

say "Building ./bin/si ..."
mkdir -p bin
go build -o bin/si ./cmd/si

say "Done. Try:"
echo "  ./bin/si --help"
echo "  ./bin/si new talk && ./bin/si play talk --open"
