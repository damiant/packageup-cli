#!/bin/sh
set -e

REPO="damiant/packageup-cli"
INSTALL_DIR="/usr/local/bin"

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *)       echo "unsupported" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)             echo "unsupported" ;;
  esac
}

OS=$(detect_os)
ARCH=$(detect_arch)

if [ "$OS" = "unsupported" ] || [ "$ARCH" = "unsupported" ]; then
  echo "Error: unsupported platform $(uname -s)/$(uname -m)"
  exit 1
fi

BASE="https://github.com/${REPO}/releases/latest/download"

echo "Installing packageup tools (${OS}/${ARCH})..."

for tool in upload download; do
  URL="${BASE}/${tool}-${OS}-${ARCH}"
  echo "  Downloading ${tool}..."
  HTTP_CODE=$(curl -sSL -o "/tmp/${tool}" -w "%{http_code}" "$URL")

  if [ "$HTTP_CODE" != "200" ]; then
    echo "Error: failed to download ${tool} (HTTP ${HTTP_CODE})"
    echo "Make sure a release exists at https://github.com/${REPO}/releases"
    exit 1
  fi

  chmod +x "/tmp/${tool}"

  if [ -w "$INSTALL_DIR" ]; then
    mv "/tmp/${tool}" "${INSTALL_DIR}/${tool}"
  else
    sudo mv "/tmp/${tool}" "${INSTALL_DIR}/${tool}"
  fi
done

echo "Installed upload and download to ${INSTALL_DIR}"
echo ""
echo "Usage:"
echo "  upload <file>              Upload a file"
echo "  download <id> [output]     Download a file by ID"
