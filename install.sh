#!/bin/bash
set -e

REPO="datsfain/kvit"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  amd64)   ARCH="amd64" ;;
  arm64)   ARCH="arm64" ;;
  aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
echo "Fetching latest version..."
VERSION=$(curl -sSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
if [ -z "$VERSION" ]; then
  echo "Error: could not determine latest version"
  exit 1
fi
echo "Latest version: $VERSION"

# Download
FILENAME="kvit_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

echo "Downloading $FILENAME..."
TMP=$(mktemp -d)
curl -sSL "$URL" -o "$TMP/$FILENAME"

# Extract
tar -xzf "$TMP/$FILENAME" -C "$TMP"

# Install
echo "Installing to $INSTALL_DIR/kvit..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/kvit" "$INSTALL_DIR/kvit"
else
  sudo mv "$TMP/kvit" "$INSTALL_DIR/kvit"
fi
chmod +x "$INSTALL_DIR/kvit"

# Cleanup
rm -rf "$TMP"

# Verify
if command -v kvit &> /dev/null; then
  echo ""
  echo "✓ kvit $VERSION installed successfully!"
  echo "  Run 'kvit --help' to get started."
else
  echo ""
  echo "✓ Installed to $INSTALL_DIR/kvit"
  echo "  If 'kvit' is not found, add $INSTALL_DIR to your PATH."
fi
