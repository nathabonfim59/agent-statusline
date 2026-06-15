#!/bin/sh
set -e

REPO="nathabonfim59/agent-statusline"
BINARY="agent-statusline"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)  ;;
    darwin) ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# detect arch
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# detect musl libc on linux
MUSL=""
if [ "$OS" = "linux" ] && ldd /bin/sh 2>&1 | grep -qi musl; then
    MUSL="-musl"
fi

# resolve version
if [ -z "$VERSION" ]; then
    VERSION=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
fi

if [ -z "$VERSION" ]; then
    echo "Could not determine latest release version"
    exit 1
fi

EXT=""
[ "$OS" = "windows" ] && EXT=".exe"

FILENAME="${BINARY}-${OS}-${ARCH}${MUSL}${EXT}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

echo "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}${MUSL}..."
curl -fsSL "$URL" -o "/tmp/${BINARY}"
chmod +x "/tmp/${BINARY}"

mkdir -p "$INSTALL_DIR"
echo "Installing to ${INSTALL_DIR}/${BINARY}..."
if [ -w "$INSTALL_DIR" ]; then
    mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "Installed ${BINARY} ${VERSION} -> ${INSTALL_DIR}/${BINARY}"

# check if install dir is in PATH
case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        echo ""
        echo "${INSTALL_DIR} is not in your PATH."
        echo "To add it, run:"
        echo ""
        echo "  echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc  # or ~/.zshrc"
        echo "  source ~/.bashrc"
        echo ""
        echo "Then restart your terminal."
        ;;
esac
