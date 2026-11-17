#!/usr/bin/env bash
# Builds the portable Windows executable (cross-compiles from Linux/macOS/Windows).
# Requires Go 1.24+. Usage: ./build.sh [version]
set -euo pipefail
cd "$(dirname "$0")"
VERSION="${1:-0.1.0}"
OUT="dist"
mkdir -p "$OUT"
go test ./internal/...
# Windows resources (icon, manifest, version info) -> .syso picked up by go build
if command -v go-winres >/dev/null 2>&1 || [ -x "$HOME/go/bin/go-winres" ]; then
  WINRES="$(command -v go-winres || echo "$HOME/go/bin/go-winres")"
  (cd cmd/smartcheck && "$WINRES" make --in winres/winres.json --arch amd64 \
     --product-version "$VERSION" --file-version "$VERSION")
fi
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath \
  -ldflags "-s -w -H windowsgui -X main.Version=$VERSION" \
  -o "$OUT/SmartMobileCheck-$VERSION-windows-x64.exe" ./cmd/smartcheck
echo "Built $OUT/SmartMobileCheck-$VERSION-windows-x64.exe"
