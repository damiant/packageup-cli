#!/bin/sh
set -e

API="https://api.packageup.io/download"

if [ -z "$1" ]; then
  echo "usage: curl -sSL https://api.packageup.io/unpack?id=FILENAME | bash"
  echo "   or: curl -sSL https://api.packageup.io/unpack | bash -s FILENAME"
  exit 1
fi

FILENAME="$1"
ARCHIVE="/tmp/packageup-$$.tar.xz"

echo "Downloading ${FILENAME}..."
HTTP_CODE=$(curl -sSL -o "$ARCHIVE" -w "%{http_code}" "${API}?filename=${FILENAME}")

if [ "$HTTP_CODE" != "200" ]; then
  echo "Error: download failed (HTTP ${HTTP_CODE})"
  rm -f "$ARCHIVE"
  exit 1
fi

echo "Extracting to current directory..."
tar -xJf "$ARCHIVE"
rm -f "$ARCHIVE"

curl -sSL -X DELETE "${API}?filename=${FILENAME}" > /dev/null

echo "${FILENAME} was unpacked"
