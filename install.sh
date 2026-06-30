#!/bin/sh
set -e

REPO="nathabonfim59/agent-statusline"
BINARY="agent-statusline"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
CONFIG_DIR="${CONFIG_DIR:-$HOME/.config/agent-statusline}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "Error: required command '$1' not found" >&2
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# Detect platform
# ---------------------------------------------------------------------------

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)  ;;
    darwin) ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# ---------------------------------------------------------------------------
# Resolve the latest v* release
# ---------------------------------------------------------------------------

VERSION="${VERSION:-}"

if [ -z "$VERSION" ] && command -v gh >/dev/null 2>&1; then
    VERSION=$(gh release view --repo "$REPO" --json tagName -q .tagName 2>/dev/null || true)
fi

if [ -z "$VERSION" ]; then
    need_cmd curl
    VERSION=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
fi

if [ -z "$VERSION" ]; then
    echo "Could not determine latest release version"
    exit 1
fi

# ---------------------------------------------------------------------------
# Download and extract the archive
# ---------------------------------------------------------------------------

EXT="tar.gz"
FILENAME="${BINARY}_${VERSION}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}..."
need_cmd curl
curl -fsSL "$URL" -o "${TMP_DIR}/${FILENAME}"

echo "Extracting ${FILENAME}..."
need_cmd tar
tar -xzf "${TMP_DIR}/${FILENAME}" -C "$TMP_DIR"

# ---------------------------------------------------------------------------
# Install binary
# ---------------------------------------------------------------------------

mkdir -p "$INSTALL_DIR"
echo "Installing to ${INSTALL_DIR}/${BINARY}..."
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi
chmod +x "${INSTALL_DIR}/${BINARY}"

# ---------------------------------------------------------------------------
# Install bundled themes and example config once
# ---------------------------------------------------------------------------

if [ -z "$NO_INSTALL_CONFIG" ] && [ ! -d "$CONFIG_DIR" ]; then
    echo "Installing default themes and example config to ${CONFIG_DIR}..."
    mkdir -p "$CONFIG_DIR"
    if [ -f "${TMP_DIR}/config.example.yaml" ]; then
        cp "${TMP_DIR}/config.example.yaml" "${CONFIG_DIR}/config.yaml"
    fi
    if [ -d "${TMP_DIR}/themes" ]; then
        cp -r "${TMP_DIR}/themes" "${CONFIG_DIR}/themes"
    fi
fi

# ---------------------------------------------------------------------------
# PATH reminder
# ---------------------------------------------------------------------------

echo "Installed ${BINARY} ${VERSION} -> ${INSTALL_DIR}/${BINARY}"

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
