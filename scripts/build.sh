#!/usr/bin/env bash
# Build ready-to-share zips for every platform into dist/.
# Usage: scripts/build.sh [version]
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
rm -rf dist && mkdir -p dist

build() { # goos goarch label
  local name="photo-select-$VERSION-$3"
  local dir="dist/$name"
  local exe="photo-select"; [ "$1" = windows ] && exe="photo-select.exe"
  mkdir -p "$dir"
  CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$dir/$exe" .
  cp README.md "$dir/"
  if [ "$1" = darwin ]; then
    # Double-clickable launcher: opens Terminal and runs the app next to it.
    printf '#!/bin/bash\ncd "$(dirname "$0")"\nxattr -dr com.apple.quarantine . 2>/dev/null\nexec ./photo-select\n' > "$dir/Start Photo Select.command"
    chmod +x "$dir/Start Photo Select.command"
  fi
  (cd dist && zip -qr "$name.zip" "$name")
  echo "built dist/$name.zip"
}

build darwin arm64 mac-apple-silicon
build darwin amd64 mac-intel
build windows amd64 windows
build linux amd64 linux
