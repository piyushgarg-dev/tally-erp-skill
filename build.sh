#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-0.1.0}"
LDFLAGS="-s -w -X main.Version=${VERSION}"
PKG="./cmd/tally"
OUTDIR="skills/tally-erp/bin"

mkdir -p "$OUTDIR"

platforms=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

for platform in "${platforms[@]}"; do
  os="${platform%/*}"
  arch="${platform#*/}"
  output="${OUTDIR}/tally-${os}-${arch}"
  if [ "$os" = "windows" ]; then
    output="${output}.exe"
  fi
  echo "Building ${output}..."
  GOOS="$os" GOARCH="$arch" go build -ldflags "$LDFLAGS" -o "$output" "$PKG"
done

echo ""
echo "Generating checksums..."
cd "$OUTDIR"
shasum -a 256 tally-* > checksums.txt
cat checksums.txt

echo ""
echo "Done. Binaries in ${OUTDIR}/:"
ls -lh tally-*
