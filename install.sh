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

# Download binary and checksums
FILENAME="kvit_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"
CHECKSUM_URL="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"

echo "Downloading $FILENAME..."
TMP=$(mktemp -d)
curl -sSL "$URL" -o "$TMP/$FILENAME"
curl -sSL "$CHECKSUM_URL" -o "$TMP/checksums.txt"

# Verify checksum
echo "Verifying checksum..."
EXPECTED=$(grep "$FILENAME" "$TMP/checksums.txt" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
  echo "Error: checksum not found for $FILENAME"
  rm -rf "$TMP"
  exit 1
fi

if command -v sha256sum &> /dev/null; then
  ACTUAL=$(sha256sum "$TMP/$FILENAME" | awk '{print $1}')
elif command -v shasum &> /dev/null; then
  ACTUAL=$(shasum -a 256 "$TMP/$FILENAME" | awk '{print $1}')
else
  echo "Warning: no sha256sum or shasum found, skipping verification"
  ACTUAL="$EXPECTED"
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Error: checksum mismatch!"
  echo "  Expected: $EXPECTED"
  echo "  Actual:   $ACTUAL"
  rm -rf "$TMP"
  exit 1
fi
echo "Checksum verified."

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

# Install shell completions
COMP_DIR=""
if [ -d "/usr/share/bash-completion/completions" ]; then
  COMP_DIR="/usr/share/bash-completion/completions"
elif [ -d "/etc/bash_completion.d" ]; then
  COMP_DIR="/etc/bash_completion.d"
fi

if [ -n "$COMP_DIR" ]; then
  echo "Installing shell completions..."
  COMP_SCRIPT=$("$INSTALL_DIR/kvit" completion bash 2>/dev/null || true)
  if [ -n "$COMP_SCRIPT" ]; then
    if [ -w "$COMP_DIR" ]; then
      echo "$COMP_SCRIPT" > "$COMP_DIR/kvit"
    else
      echo "$COMP_SCRIPT" | sudo tee "$COMP_DIR/kvit" > /dev/null
    fi
  fi
fi

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
