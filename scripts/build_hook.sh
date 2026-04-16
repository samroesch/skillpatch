#!/usr/bin/env bash
# Build the skill broker hook binary for all platforms.
# Output lands in broker-plugin/hooks/
# Run from the repo root: bash scripts/build_hook.sh

set -euo pipefail

SRC="broker-plugin/hooks/prompt_broker.go"
OUT="broker-plugin/hooks"
LDFLAGS="-s -w"

echo "Building skill broker hook..."

GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$OUT/prompt_broker_windows_amd64.exe" "$SRC"
echo "  ✓ windows/amd64"

GOOS=darwin  GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$OUT/prompt_broker_darwin_amd64"       "$SRC"
echo "  ✓ darwin/amd64"

GOOS=darwin  GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "$OUT/prompt_broker_darwin_arm64"       "$SRC"
echo "  ✓ darwin/arm64 (Apple Silicon)"

GOOS=linux   GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$OUT/prompt_broker_linux_amd64"        "$SRC"
echo "  ✓ linux/amd64"

GOOS=linux   GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "$OUT/prompt_broker_linux_arm64"        "$SRC"
echo "  ✓ linux/arm64"

echo ""
echo "Done. Sizes:"
ls -lh "$OUT"/prompt_broker_* | awk '{print "  " $5 "  " $9}'
