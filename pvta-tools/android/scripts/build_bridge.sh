#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
ANDROID_DIR="$ROOT_DIR/android"
BRIDGE_DIR="$ANDROID_DIR/go-bridge"
OUTPUT_AAR="$ANDROID_DIR/app/libs/pvta-mobilebridge.aar"

mkdir -p "$(dirname "$OUTPUT_AAR")"

if ! command -v gomobile >/dev/null 2>&1; then
  echo "gomobile not found in PATH" >&2
  exit 1
fi

if ! command -v gobind >/dev/null 2>&1; then
  echo "gobind not found in PATH" >&2
  exit 1
fi

export ANDROID_HOME="${ANDROID_HOME:-$HOME/Library/Android/sdk}"
export ANDROID_SDK_ROOT="${ANDROID_SDK_ROOT:-$ANDROID_HOME}"
export GOCACHE="${GOCACHE:-/tmp/pvta-go-build}"
export GOMOBILE="${GOMOBILE:-/tmp/gomobile}"

cd "$BRIDGE_DIR"
gomobile bind \
  -target=android \
  -androidapi 24 \
  -javapkg=com.vibrazioni.pvta.bridge \
  -o "$OUTPUT_AAR" \
  .

echo "Wrote $OUTPUT_AAR"
