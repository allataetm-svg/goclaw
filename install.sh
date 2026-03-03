#!/bin/bash

# GoClaw Clean Install Script

set -e

echo "🦞 GoClaw Kurulum Scripti"

# Platform detection
ARCH=$(uname -m)
OS=$(uname -s)

case "$OS" in
    Linux)
        case "$ARCH" in
            x86_64|amd64) PLATFORM="linux-amd64" ;;
            aarch64|arm64) PLATFORM="linux-arm64" ;;
            *) echo "Desteklenmeyen mimari: $ARCH"; exit 1 ;;
        esac
        ;;
    Darwin)
        case "$ARCH" in
            x86_64|amd64) PLATFORM="darwin-amd64" ;;
            aarch64|arm64) PLATFORM="darwin-arm64" ;;
            *) echo "Desteklenmeyen mimari: $ARCH"; exit 1 ;;
        esac
        ;;
    *)
        echo "Desteklenmeyen işletim sistemi: $OS"
        exit 1
        ;;
esac

echo "Platform: $PLATFORM"

# Remove old goclaw
echo "Eski goclaw kaldırılıyor..."
sudo rm -f /usr/local/bin/goclaw 2>/dev/null || true
rm -f ./goclaw 2>/dev/null || true

# Download new binary
echo "Yeni goclaw indiriliyor..."
curl -L -o goclaw "https://raw.githubusercontent.com/allataetm-svg/goclaw/feature/enhanced-memory-system/goclaw-${PLATFORM}"

# Make executable
chmod +x goclaw

# Install
echo "Kuruluyor..."
sudo mv goclaw /usr/local/bin/goclaw

# Verify
if goclaw --help > /dev/null 2>&1; then
    echo "✅ GoClaw başarıyla kuruldu!"
    goclaw --help
else
    echo "❌ Kurulum başarısız"
    exit 1
fi
