#!/bin/sh
# Capy installer. Downloads the latest release for the current OS/arch
# from https://github.com/olivierdevelops/capy/releases and installs into a
# directory on $PATH.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/olivierdevelops/capy/main/scripts/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/olivierdevelops/capy/main/scripts/install.sh | sh -s -- --version v0.1.0
#   curl -fsSL https://raw.githubusercontent.com/olivierdevelops/capy/main/scripts/install.sh | sh -s -- --dir ~/.local/bin

set -e

REPO="olivierdevelops/capy"
VERSION="latest"
INSTALL_DIR=""

while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --dir)     INSTALL_DIR="$2"; shift 2 ;;
    *) printf "unknown flag: %s\n" "$1"; exit 1 ;;
  esac
done

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS=linux ;;
  darwin) OS=darwin ;;
  *) printf "unsupported OS: %s\n" "$OS"; exit 1 ;;
esac

# Detect ARCH
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) printf "unsupported arch: %s\n" "$ARCH"; exit 1 ;;
esac

# Resolve latest version if needed
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name":' \
    | head -1 \
    | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "$VERSION" ]; then
    printf "could not resolve latest version\n" >&2
    exit 1
  fi
fi

# Strip leading 'v' for the archive name; keep tag form for URL
VERSION_NUM=$(printf "%s" "$VERSION" | sed -e 's/^v//')

ARCHIVE="capy_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

printf "downloading %s\n" "$URL"
TMP=$(mktemp -d)
trap "rm -rf $TMP" EXIT

curl -fsSL -o "$TMP/$ARCHIVE" "$URL"

# Verify checksum if possible
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
if curl -fsSL -o "$TMP/checksums.txt" "$CHECKSUM_URL" 2>/dev/null; then
  ACTUAL=$(sha256sum "$TMP/$ARCHIVE" 2>/dev/null | awk '{print $1}' \
    || shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')
  EXPECTED=$(grep "$ARCHIVE" "$TMP/checksums.txt" | awk '{print $1}')
  if [ "$ACTUAL" != "$EXPECTED" ]; then
    printf "checksum mismatch for %s\n" "$ARCHIVE" >&2
    exit 1
  fi
  printf "checksum OK\n"
fi

# Extract
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"

# Decide install dir
if [ -z "$INSTALL_DIR" ]; then
  if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi
fi

mv "$TMP/capy" "$INSTALL_DIR/capy"
chmod +x "$INSTALL_DIR/capy"

printf "installed %s/capy\n" "$INSTALL_DIR"
"$INSTALL_DIR/capy" version || true

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    printf "\nNote: %s is not on PATH. Add this to your shell rc:\n" "$INSTALL_DIR"
    printf "  export PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR"
    ;;
esac
