#!/bin/sh
set -e

# actionlint-mcp installer script
# Usage: curl -sSfL https://raw.githubusercontent.com/hongkongkiwi/actionlint-mcp/main/install.sh | sh -s -- -b /usr/local/bin

BINARY_NAME="actionlint-mcp"
GITHUB_REPO="hongkongkiwi/actionlint-mcp"

usage() {
    cat <<EOF
Usage: $0 [-b BIN_DIR] [-v VERSION]
  -b BIN_DIR    Install directory (default: /usr/local/bin)
  -v VERSION    Version to install (default: latest)
  -h            Show this help message

Examples:
  # Install latest version to /usr/local/bin
  $0 -b /usr/local/bin

  # Install specific version
  $0 -v v1.0.0

  # Install to custom directory
  $0 -b ~/bin
EOF
    exit 0
}

# Default values
BIN_DIR="/usr/local/bin"
VERSION="latest"

# Parse arguments
while getopts "b:v:h" opt; do
    case $opt in
        b) BIN_DIR="$OPTARG" ;;
        v) VERSION="$OPTARG" ;;
        h) usage ;;
        *) usage ;;
    esac
done

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
case $ARCH in
    x86_64) ARCH="x86_64" ;;
    amd64) ARCH="x86_64" ;;
    aarch64) ARCH="arm64" ;;
    arm64) ARCH="arm64" ;;
    armv7l) ARCH="armv7" ;;
    armv6l) ARCH="armv6" ;;
    i386) ARCH="i386" ;;
    i686) ARCH="i386" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Map OS names
case $OS in
    linux) OS="Linux" ;;
    darwin) OS="Darwin" ;;
    freebsd) OS="FreeBSD" ;;
    openbsd) OS="OpenBSD" ;;
    netbsd) OS="NetBSD" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest version if not specified
if [ "$VERSION" = "latest" ]; then
    echo "Fetching latest version..."
    VERSION=$(curl -sSfL "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "Failed to fetch latest version"
        exit 1
    fi
fi

# Remove 'v' prefix if present
VERSION_NUM=${VERSION#v}

# Construct download URL
FILENAME="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
if [ "$OS" = "Windows" ]; then
    FILENAME="${BINARY_NAME}_${OS}_${ARCH}.zip"
fi

DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION/$FILENAME"
CHECKSUMS_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION/checksums.txt"

echo "Installing $BINARY_NAME $VERSION for $OS/$ARCH..."
echo "Download URL: $DOWNLOAD_URL"

# Create temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Download binary archive
echo "Downloading $BINARY_NAME..."
if ! curl -sSfL "$DOWNLOAD_URL" -o "$TMP_DIR/$FILENAME"; then
    echo "Failed to download $BINARY_NAME"
    exit 1
fi

# Download and verify checksums
echo "Downloading checksums..."
if curl -sSfL "$CHECKSUMS_URL" -o "$TMP_DIR/checksums.txt" 2>/dev/null; then
    echo "Verifying checksum..."
    cd "$TMP_DIR"
    
    # Extract checksum for our file
    EXPECTED_CHECKSUM=$(grep "$FILENAME" checksums.txt | awk '{print $1}')
    
    if [ -n "$EXPECTED_CHECKSUM" ]; then
        # Calculate actual checksum
        if command -v sha256sum >/dev/null 2>&1; then
            ACTUAL_CHECKSUM=$(sha256sum "$FILENAME" | awk '{print $1}')
        elif command -v shasum >/dev/null 2>&1; then
            ACTUAL_CHECKSUM=$(shasum -a 256 "$FILENAME" | awk '{print $1}')
        else
            echo "Warning: Cannot verify checksum (sha256sum or shasum not found)"
            ACTUAL_CHECKSUM=""
        fi
        
        if [ -n "$ACTUAL_CHECKSUM" ]; then
            if [ "$EXPECTED_CHECKSUM" = "$ACTUAL_CHECKSUM" ]; then
                echo "✅ Checksum verified"
            else
                echo "❌ Checksum verification failed!"
                echo "Expected: $EXPECTED_CHECKSUM"
                echo "Actual:   $ACTUAL_CHECKSUM"
                exit 1
            fi
        fi
    fi
else
    echo "Warning: Could not download checksums file"
fi

# Extract archive
echo "Extracting archive..."
if [ "$OS" = "Windows" ]; then
    unzip -q "$TMP_DIR/$FILENAME" -d "$TMP_DIR"
else
    tar -xzf "$TMP_DIR/$FILENAME" -C "$TMP_DIR"
fi

# Find the binary
BINARY_PATH="$TMP_DIR/$BINARY_NAME"
if [ ! -f "$BINARY_PATH" ]; then
    # Binary might be in a subdirectory
    BINARY_PATH=$(find "$TMP_DIR" -name "$BINARY_NAME" -type f | head -n 1)
    if [ -z "$BINARY_PATH" ]; then
        echo "Binary not found in archive"
        exit 1
    fi
fi

# Create bin directory if it doesn't exist
if [ ! -d "$BIN_DIR" ]; then
    echo "Creating directory: $BIN_DIR"
    mkdir -p "$BIN_DIR"
fi

# Install binary
echo "Installing to $BIN_DIR/$BINARY_NAME..."
if ! mv "$BINARY_PATH" "$BIN_DIR/$BINARY_NAME"; then
    echo "Failed to install binary (try using sudo)"
    exit 1
fi

# Make executable
chmod +x "$BIN_DIR/$BINARY_NAME"

# Verify installation
if "$BIN_DIR/$BINARY_NAME" -version >/dev/null 2>&1; then
    echo "✅ Successfully installed $BINARY_NAME $VERSION to $BIN_DIR"
    echo ""
    "$BIN_DIR/$BINARY_NAME" -version
else
    echo "⚠️  Installation completed but could not verify binary"
fi

# Check if BIN_DIR is in PATH
case ":$PATH:" in
    *":$BIN_DIR:"*)
        ;;
    *)
        echo ""
        echo "⚠️  Note: $BIN_DIR is not in your PATH"
        echo "Add it to your PATH by running:"
        echo "  export PATH=\"$BIN_DIR:\$PATH\""
        ;;
esac