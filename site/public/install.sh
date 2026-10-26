#!/usr/bin/env bash
# Photo Select installer for macOS and Linux.
# Usage: curl -fsSL https://<site>/install.sh | bash -s https://<site>
# Installs into ~/PhotoSelect, adds a Desktop launcher on macOS, then starts the app.
# Files fetched with curl aren't quarantined, so macOS Gatekeeper doesn't block it.
set -euo pipefail

main() {
  local base="${1:-${PHOTO_SELECT_URL:-}}"
  if [ -z "$base" ]; then
    echo "Copy the full command from the Photo Select website and paste it again." >&2
    exit 1
  fi
  base="${base%/}"
  local dest="$HOME/PhotoSelect" label

  case "$(uname -s)" in
    Darwin)
      if [ "$(sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" = 1 ]; then
        label=mac-apple-silicon
      else
        label=mac-intel
      fi ;;
    Linux)
      case "$(uname -m)" in
        x86_64 | amd64) label=linux ;;
        aarch64 | arm64) label=linux-arm64 ;;
        *) echo "Sorry, this computer ($(uname -m)) isn't supported." >&2; exit 1 ;;
      esac ;;
    *)
      echo "On Windows, use the PowerShell command from the website instead." >&2
      exit 1 ;;
  esac

  local tmp
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  echo ""
  echo "  📸  Downloading Photo Select ($label)…"
  curl -fL --progress-bar "$base/downloads/photo-select-$label.zip" -o "$tmp/app.zip"
  unzip -q "$tmp/app.zip" -d "$tmp/x"
  local bin
  bin="$(find "$tmp/x" -type f -name photo-select | head -n 1)"
  if [ -z "$bin" ]; then
    echo "The download looks broken. Please try again." >&2
    exit 1
  fi

  mkdir -p "$dest"
  # Copy next to the old binary then rename, so a running copy isn't disturbed.
  cp "$bin" "$dest/photo-select.new"
  chmod +x "$dest/photo-select.new"
  mv -f "$dest/photo-select.new" "$dest/photo-select"

  local again="$dest/photo-select"
  if [ "$label" = mac-apple-silicon ] || [ "$label" = mac-intel ]; then
    local launcher="$HOME/Desktop/Photo Select.command"
    printf '#!/bin/bash\nexec "$HOME/PhotoSelect/photo-select"\n' > "$launcher"
    chmod +x "$launcher"
    again="double-click “Photo Select” on your Desktop"
  fi

  echo ""
  echo "  ✅  Installed in $dest"
  echo "  Next time: $again"
  echo ""
  echo "  Starting it now. Your browser will open the setup page."
  echo "  If your Mac asks to accept incoming network connections, click Allow."
  echo ""
  cd "$dest"
  exec ./photo-select </dev/null
}

main "$@"
