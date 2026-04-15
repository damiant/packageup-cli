#!/bin/sh
set -e

API="https://api.packageup.io/upload"
ARCHIVE="/tmp/packageup-$$.tar.xz"

tar -cJf "$ARCHIVE" -C . .

RESPONSE=$(curl -sSL -X POST -H "Content-Type: application/octet-stream" --data-binary "@${ARCHIVE}" "$API")
rm -f "$ARCHIVE"

FILENAME=$(echo "$RESPONSE" | grep -o '"filename":"[^"]*"' | cut -d'"' -f4)

if [ -z "$FILENAME" ]; then
  echo "Error: upload failed"
  echo "$RESPONSE"
  exit 1
fi

echo "${FILENAME} was created"
echo ""
echo "To unpack, run:"
echo "  curl -sSL https://api.packageup.io/unpack | bash -s ${FILENAME}"
