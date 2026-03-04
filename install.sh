#!/bin/bash

set -e

echo "🦞 GoClaw Installer"
echo "===================="

OS=$(uname -s)
ARCH=$(uname -m)

echo "Detected: $OS ($ARCH)"

if ! command -v go &> /dev/null; then
    echo "Installing Go..."
    if command -v apt-get &> /dev/null; then
        sudo apt-get update && sudo apt-get install -y golang-go
    elif command -v dnf &> /dev/null; then
        sudo dnf install -y golang
    elif command -v yum &> /dev/null; then
        sudo yum install -y golang
    elif command -v brew &> /dev/null; then
        brew install go
    elif command -v pacman &> /dev/null; then
        sudo pacman -S --noconfirm go
    else
        echo "No package manager. Install Go manually: https://go.dev/dl/"
        exit 1
    fi
fi

echo "Go version: $(go version)"

mkdir -p "$HOME/.goclaw"

if [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo "Installing to: $INSTALL_DIR"

SCRIPT_DIR="$(cd "$(dirname "$0")" 2>/dev/null && pwd)"

if [ -d "$SCRIPT_DIR/.git" ]; then
    cd "$SCRIPT_DIR"
    GOOS=$OS GOARCH=$ARCH go build -o "$INSTALL_DIR/goclaw" .
else
    echo "Downloading latest release..."
    DOWNLOAD_URL=$(curl -sL "https://api.github.com/repos/allataetm-svg/goclaw/releases/latest" | grep -o "browser_download_url.*goclaw-${OS}-${ARCH}" | head -1 | cut -d'"' -f4)
    if [ -n "$DOWNLOAD_URL" ]; then
        curl -sL "$DOWNLOAD_URL" -o "$INSTALL_DIR/goclaw"
        chmod +x "$INSTALL_DIR/goclaw"
    else
        echo "No prebuilt binary for $OS-$ARCH. Building from source requires git clone."
        exit 1
    fi
fi

export PATH="$PATH:$INSTALL_DIR"

echo ""
echo "========================================"
echo "✅ GoClaw installed successfully!"
echo "========================================"
echo ""
echo "Next steps:"
echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
echo "  goclaw onboard"
echo ""
