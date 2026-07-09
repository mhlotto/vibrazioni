#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
Usage:
  deploy-kiln-zip.sh SITE.zip USER HOST REMOTE_PATH

Example:
  ./deploy-kiln-zip.sh amherst-directory.zip arr example.com /var/www/amherst-directory

What it does:
  - verifies SITE.zip exists locally
  - copies SITE.zip to USER@HOST
  - creates REMOTE_PATH if needed
  - unzips SITE.zip into REMOTE_PATH
  - overwrites files that are in the zip
  - does not delete REMOTE_PATH first
  - removes the uploaded zip from the remote temp location
EOF
}

if [ "$#" -ne 4 ]; then
  usage
  exit 2
fi

ZIP_PATH=$1
REMOTE_USER=$2
REMOTE_HOST=$3
REMOTE_PATH=$4

if [ ! -f "$ZIP_PATH" ]; then
  echo "error: zip file not found: $ZIP_PATH" >&2
  exit 1
fi

case "$ZIP_PATH" in
  *.zip) ;;
  *)
    echo "error: input file must end in .zip: $ZIP_PATH" >&2
    exit 1
    ;;
esac

ZIP_BASENAME=$(basename "$ZIP_PATH")
REMOTE_TMP="/tmp/kiln-deploy-${USER:-user}-$$-${ZIP_BASENAME}"

echo "Copying $ZIP_PATH to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TMP}"
scp "$ZIP_PATH" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TMP}"

echo "Deploying to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PATH}"
ssh "${REMOTE_USER}@${REMOTE_HOST}" 'sh -s' -- "$REMOTE_TMP" "$REMOTE_PATH" <<'REMOTE_SCRIPT'
set -eu

ZIP_FILE=$1
TARGET_DIR=$2

if [ ! -f "$ZIP_FILE" ]; then
  echo "error: uploaded zip not found on remote: $ZIP_FILE" >&2
  exit 1
fi

mkdir -p "$TARGET_DIR"

# unzip -o overwrites files from the zip, but does not delete unrelated files.
unzip -o "$ZIP_FILE" -d "$TARGET_DIR"

rm -f "$ZIP_FILE"

echo "Deployed zip contents to $TARGET_DIR"
REMOTE_SCRIPT

echo "Done."
