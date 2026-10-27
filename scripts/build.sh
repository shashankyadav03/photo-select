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
  if [ "$1" = darwin ]; then
    # A real double-clickable app: no Terminal, own icon, opens the browser itself.
    local app="$dir/Photo Select.app/Contents"
    mkdir -p "$app/MacOS" "$app/Resources"
    CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$app/MacOS/photo-select" .
    cp packaging/mac/icon.icns "$app/Resources/icon.icns"
    sed "s/@VERSION@/${VERSION#v}/g" packaging/mac/Info.plist.tmpl > "$app/Info.plist"
    # Seal the bundle (ad-hoc). Without this Gatekeeper calls the app "damaged" instead of
    # offering "Open Anyway". codesign only exists on macOS, so release builds run on macOS.
    if command -v codesign >/dev/null; then
      codesign --force --deep -s - "$dir/Photo Select.app"
    else
      echo "warning: codesign not found; the Mac app will be flagged as damaged. Build on macOS." >&2
    fi
    # Zip only the app, so unzipping leaves "Photo Select" ready to double-click.
    (cd "$dir" && zip -qry "../$name.zip" "Photo Select.app")
  else
    CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$dir/$exe" .
    cp README.md "$dir/"
    (cd dist && zip -qr "$name.zip" "$name")
  fi
  echo "built dist/$name.zip"
}

build darwin arm64 mac-apple-silicon
build darwin amd64 mac-intel
build windows amd64 windows
build linux amd64 linux
build linux arm64 linux-arm64
