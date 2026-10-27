#!/usr/bin/env bash
# Build the friends homepage (site/) with the app zips bundled in, ready to upload.
# Usage: scripts/build-site.sh [version]   → site/dist/
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"

scripts/build.sh "$VERSION"

# Stable names (photo-select-<platform>.zip) so the installers and links never change.
out=site/public/downloads
rm -rf "$out" && mkdir -p "$out"
files=""
for zip in dist/photo-select-"$VERSION"-*.zip; do
  label="${zip#dist/photo-select-"$VERSION"-}"
  label="${label%.zip}"
  cp "$zip" "$out/photo-select-$label.zip"
  size=$(wc -c < "$zip" | tr -d ' ')
  files="$files${files:+,}\"$label\":{\"name\":\"photo-select-$label.zip\",\"size\":$size}"
done
printf '{"version":"%s","files":{%s}}\n' "$VERSION" "$files" > "$out/manifest.json"

(cd site && npm ci --no-audit --no-fund && npm run build)
echo "site ready in site/dist/"
