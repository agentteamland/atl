#!/usr/bin/env sh
# One-liner installer for atl.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
#
# Env overrides:
#   ATL_VERSION=v2.0.0               specific release (defaults to latest)
#   ATL_INSTALL_DIR=/usr/local/bin   where to put the binary
#
set -eu

REPO="agentteamland/atl"
BINARY_NAME="atl"
INSTALL_DIR="${ATL_INSTALL_DIR:-/usr/local/bin}"
VERSION="${ATL_VERSION:-}"

# Detect OS.
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux | darwin) ;;
  *)
    echo "Unsupported OS: $OS (supported: linux, darwin)"
    exit 1
    ;;
esac

# Detect arch.
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64) ARCH=amd64 ;;
  arm64 | aarch64) ARCH=arm64 ;;
  *)
    echo "Unsupported arch: $ARCH (supported: amd64, arm64)"
    exit 1
    ;;
esac

# Resolve latest version if not pinned.
if [ -z "$VERSION" ]; then
  echo "→ Resolving latest release..."
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
    sed -n 's/.*"tag_name": *"\(v[^"]*\)".*/\1/p' | head -n1)"
  if [ -z "$VERSION" ]; then
    echo "Could not resolve latest version. Set ATL_VERSION=vX.Y.Z manually."
    exit 1
  fi
fi

VERSION_NO_V="${VERSION#v}"
ARCHIVE="atl_${VERSION_NO_V}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo "→ Downloading ${URL}"
TMP="$(mktemp -d)"
cd "$TMP"

curl -fsSL -o "$ARCHIVE" "$URL"
tar xzf "$ARCHIVE"

if [ ! -f "$BINARY_NAME" ]; then
  echo "Extracted archive did not contain ${BINARY_NAME}."
  exit 1
fi

chmod +x "$BINARY_NAME"

# Install: need sudo if install dir is system-owned.
if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY_NAME" "${INSTALL_DIR}/${BINARY_NAME}"
else
  echo "→ Installing to ${INSTALL_DIR} (may require sudo)"
  sudo mv "$BINARY_NAME" "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo ""
echo "✓ atl ${VERSION} installed to ${INSTALL_DIR}/${BINARY_NAME}"
"${INSTALL_DIR}/${BINARY_NAME}" --version
echo ""
echo "Next: cd into a project and run:"
echo "  atl install agentteamland/software-project-team"
