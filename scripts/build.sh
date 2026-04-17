#!/usr/bin/env bash
# Build all skillpatch binaries for all platforms.
# Run from the repo root: bash scripts/build.sh
#
# Outputs:
#   broker-plugin/mcp/skill_server_*   MCP server (replaces hook)

set -euo pipefail

MCP_SRC="broker-plugin/mcp/skill_server.go"
MCP_OUT="broker-plugin/mcp"
LDFLAGS="-s -w"

echo "Building MCP skill server..."

GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$MCP_OUT/skill_server_windows_amd64.exe" "$MCP_SRC"
echo "  ✓ windows/amd64"

GOOS=darwin  GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$MCP_OUT/skill_server_darwin_amd64"       "$MCP_SRC"
echo "  ✓ darwin/amd64"

GOOS=darwin  GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "$MCP_OUT/skill_server_darwin_arm64"       "$MCP_SRC"
echo "  ✓ darwin/arm64 (Apple Silicon)"

GOOS=linux   GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$MCP_OUT/skill_server_linux_amd64"        "$MCP_SRC"
echo "  ✓ linux/amd64"

GOOS=linux   GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "$MCP_OUT/skill_server_linux_arm64"        "$MCP_SRC"
echo "  ✓ linux/arm64"

echo ""
echo "Done. Sizes:"
ls -lh "$MCP_OUT"/skill_server_* | awk '{print "  " $5 "  " $9}'
