#!/bin/sh
set -e

# printctl One-Click Installer
# Installs printctl binary to /usr/local/bin or ~/.local/bin

REPO="boheastill/printctl"
TAG="v0.0.1-alpha"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    *) echo "Unsupported operating system: $OS"; exit 1 ;;
esac

FILENAME="printctl_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${FILENAME}"

INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo "[*] Downloading printctl ${TAG} for ${OS}/${ARCH}..."
TMP_DIR="$(mktemp -d)"
curl -fsSL "$URL" -o "${TMP_DIR}/${FILENAME}"

echo "[*] Extracting..."
tar -xzf "${TMP_DIR}/${FILENAME}" -C "$TMP_DIR"

echo "[*] Installing printctl to ${INSTALL_DIR}..."
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMP_DIR}/printctl" "${INSTALL_DIR}/printctl"
    chmod +x "${INSTALL_DIR}/printctl"
else
    sudo mv "${TMP_DIR}/printctl" "${INSTALL_DIR}/printctl"
    sudo chmod +x "${INSTALL_DIR}/printctl"
fi

rm -rf "$TMP_DIR"

echo "[+] printctl successfully installed!"
echo "[+] Run 'printctl discover' to find your network printers."
