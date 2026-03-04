#!/bin/bash

set -e

echo "🦞 GoClaw Installer"
echo "===================="

# Detect OS and Arch
OS=$(uname -s)
ARCH=$(uname -m)

echo "Detected: $OS ($ARCH)"

# Determine package manager
if command -v apt-get &> /dev/null; then
    PKG_MGR="apt"
elif command -v dnf &> /dev/null; then
    PKG_MGR="dnf"
elif command -v yum &> /dev/null; then
    PKG_MGR="yum"
elif command -v brew &> /dev/null; then
    PKG_MGR="brew"
elif command -v pacman &> /dev/null; then
    PKG_MGR="pacman"
else
    PKG_MGR="none"
fi

echo "Package manager: $PKG_MGR"

# Install Go if not present
if ! command -v go &> /dev/null; then
    echo "Installing Go..."
    case $PKG_MGR in
        apt)
            sudo apt-get update
            sudo apt-get install -y golang-go
            ;;
        dnf)
            sudo dnf install -y golang
            ;;
        yum)
            sudo yum install -y golang
            ;;
        brew)
            brew install go
            ;;
        pacman)
            sudo pacman -S --noconfirm go
            ;;
        none)
            echo "No package manager found. Please install Go manually: https://go.dev/dl/"
            exit 1
            ;;
    esac
fi

echo "Go version: $(go version)"

# Create config directory
CONFIG_DIR="$HOME/.goclaw"
mkdir -p "$CONFIG_DIR"

# Detect install location
if [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
elif [ -w "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo "Installing to: $INSTALL_DIR"

# Build from source
echo "Building GoClaw..."
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ -d "$SCRIPT_DIR/.git" ]; then
    cd "$SCRIPT_DIR"
    GOOS=$OS GOARCH=$ARCH go build -o "$INSTALL_DIR/goclaw" .
else
    # Download latest release
    echo "Downloading latest release..."
    curl -sL "https://api.github.com/repos/allataetm-svg/goclaw/releases/latest" | grep -o '"browser_download_url": *"[^"]*'"${OS}"'-'"${ARCH}"'" | cut -d'"' -f4 | xargs -I {} curl -sL -o "$INSTALL_DIR/goclaw" {}
    chmod +x "$INSTALL_DIR/goclaw"
fi

# Add to PATH
SHELL_RC="$HOME/.bashrc"
[ -f "$HOME/.zshrc" ] && SHELL_RC="$HOME/.zshrc"
[ -f "$HOME/.profile" ] && SHELL_RC="$HOME/.profile"

if ! grep -q "goclaw" "$SHELL_RC" 2>/dev/null; then
    echo 'export PATH="$PATH:$HOME/.local/bin"' >> "$SHELL_RC"
    echo "Added to PATH in $SHELL_RC"
fi

echo ""
echo "✅ GoClaw installed!"
echo ""
echo "Next steps:"
echo "  source $SHELL_RC"
echo "  goclaw onboard"
echo ""
