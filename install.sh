#!/usr/bin/env sh
# Install tokentop from the latest GitHub release.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/lwlee2608/tokentop/main/install.sh | sh
#
# Env vars:
#   VERSION   release tag to install (default: latest)
#   PREFIX    install prefix (default: $HOME/.local)

set -eu

REPO="lwlee2608/tokentop"
APP="tokentop"
PREFIX="${PREFIX:-$HOME/.local}"
VERSION="${VERSION:-}"

err() { printf 'error: %s\n' "$*" >&2; exit 1; }

need() { command -v "$1" >/dev/null 2>&1 || err "missing required command: $1"; }

need curl
need tar
need uname
need install

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux|darwin) ;;
  *) err "unsupported OS: $os" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) err "unsupported architecture: $arch" ;;
esac

if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
    | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)
  [ -n "$VERSION" ] || err "failed to resolve latest release tag"
fi

# Archive uses version without leading 'v'.
ver_noprefix=${VERSION#v}
archive="${APP}_${ver_noprefix}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$VERSION/$archive"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

printf 'Downloading %s\n' "$url"
curl -fsSL "$url" -o "$tmp/$archive" || err "download failed: $url"

tar -xzf "$tmp/$archive" -C "$tmp"
[ -f "$tmp/$APP" ] || err "binary '$APP' not found in archive"

install -d "$PREFIX/bin"
install -m 755 "$tmp/$APP" "$PREFIX/bin/$APP"

printf 'Installed %s %s to %s/bin/%s\n' "$APP" "$VERSION" "$PREFIX" "$APP"

case ":$PATH:" in
  *":$PREFIX/bin:"*) ;;
  *) printf 'Note: %s/bin is not in your PATH.\n' "$PREFIX" ;;
esac
