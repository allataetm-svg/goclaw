#!/bin/bash

# GoClaw Clean Install Script

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

# Detect install location
if [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
    SUDO=""
elif [ -w "$HOME/bin" ]; then
    INSTALL_DIR="$HOME/bin"
    SUDO=""
else
    INSTALL_DIR="/usr/local/bin"
    SUDO="sudo"
fi

echo "Kurulum dizini: $INSTALL_DIR"

# Remove old goclaw
echo "Eski goclaw kaldırılıyor..."
rm -f "$INSTALL_DIR/goclaw" 2>/dev/null || true
rm -f ./goclaw 2>/dev/null || true

# Download new binary
echo "Yeni goclaw indiriliyor..."
URL="https://raw.githubusercontent.com/allataetm-svg/goclaw/feature/enhanced-memory-system/goclaw-${PLATFORM}"
echo "URL: $URL"

if ! curl -fSL -o goclaw "$URL"; then
    echo "❌ İndirme başarısız!"
    echo "Alternatif olarak şunu deneyin:"
    echo "  wget -O goclaw '$URL'"
    exit 1
fi

# Make executable
chmod +x goclaw

# Verify file
echo "İndirilen dosya kontrol ediliyor..."
if [ ! -s goclaw ]; then
    echo "❌ İndirilen dosya boş veya hatalı"
    exit 1
fi

FILE_TYPE=$(file goclaw)
echo "Dosya tipi: $FILE_TYPE"

# Install
echo "Kuruluyor..."
if ! $SUDO mv goclaw "$INSTALL_DIR/goclaw"; then
    echo "❌ Kurulum başarısız - izinleri kontrol edin"
    exit 1
fi

# Verify
echo "Doğrulanıyor..."
if "$INSTALL_DIR/goclaw" --help > /dev/null 2>&1; then
    echo "✅ GoClaw başarıyla kuruldu!"
    echo ""
    "$INSTALL_DIR/goclaw" --help
else
    echo "❌ Doğrulama başarısız"
    echo "Dosya tipi: $(file $INSTALL_DIR/goclaw)"
    exit 1
fi
