#!/usr/bin/env bash

set -euo pipefail

INPUT=""
OUTPUT=""
CUT_LENGTH=""

# ---------------------------
# Parse CLI args
# ---------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --input)
            INPUT="$2"
            shift 2
            ;;
        --output)
            OUTPUT="$2"
            shift 2
            ;;
        --cut-length)
            CUT_LENGTH="$2"
            shift 2
            ;;
        *)
            echo "Unknown argument: $1"
            exit 1
            ;;
    esac
done

if [[ -z "$INPUT" || -z "$OUTPUT" || -z "$CUT_LENGTH" ]]; then
    echo "Usage: $0 --input <video> --output <video> --cut-length <seconds>"
    exit 1
fi

TMP_DIR=$(mktemp -d)
SEG_DIR="$TMP_DIR/segs"
mkdir -p "$SEG_DIR"

echo "Temp dir: $TMP_DIR"

# ---------------------------
# Get duration
# ---------------------------
DURATION=$(ffprobe -v error -show_entries format=duration \
    -of default=noprint_wrappers=1:nokey=1 "$INPUT")

DURATION_INT=${DURATION%.*}

NUM_SEGMENTS=$(( DURATION_INT / CUT_LENGTH ))

echo "Duration: $DURATION_INT seconds"
echo "Segments: $NUM_SEGMENTS"

# ---------------------------
# Cut segments
# ---------------------------
for ((i=0;i<NUM_SEGMENTS;i++)); do
    START=$(( i * CUT_LENGTH ))
    OUTFILE=$(printf "%s/seg_%05d.mp4" "$SEG_DIR" "$i")

    ffmpeg -loglevel error -y \
        -ss "$START" \
        -i "$INPUT" \
        -t "$CUT_LENGTH" \
        -c copy \
        "$OUTFILE"
done

# ---------------------------
# Shuffle
# ---------------------------
ls "$SEG_DIR"/seg_*.mp4 | shuf > "$TMP_DIR/shuffled.txt"

# ---------------------------
# Build concat list
# ---------------------------
> "$TMP_DIR/concat.txt"

while read -r f; do
    echo "file '$f'" >> "$TMP_DIR/concat.txt"
done < "$TMP_DIR/shuffled.txt"

# ---------------------------
# Concatenate
# ---------------------------
ffmpeg -loglevel error -y \
    -f concat \
    -safe 0 \
    -i "$TMP_DIR/concat.txt" \
    -c copy \
    "$OUTPUT"

echo "Output written to: $OUTPUT"

rm -rf "$TMP_DIR"
