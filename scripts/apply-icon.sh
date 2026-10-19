#!/bin/bash
# Applies a newly exported 1024x1024 icon to mygou.app
set -euo pipefail
cd "$(dirname "$0")/.."

SRC="${1:-}"

if [[ -z "$SRC" ]]; then
  if [[ -f "$HOME/Downloads/mygou-icon-1024.png" ]]; then
    SRC="$HOME/Downloads/mygou-icon-1024.png"
  elif [[ -f "build/icon-1024.png" ]]; then
    SRC="build/icon-1024.png"
  elif [[ -f ".tmp/apple_liquid_pro.png" ]]; then
    SRC=".tmp/apple_liquid_pro.png"
  else
    echo "Usage: ./scripts/apply-icon.sh <path-to-1024x1024.png>"
    echo "Or place mygou-icon-1024.png in ~/Downloads and rerun."
    exit 1
  fi
fi

echo "Applying icon from: $SRC"
mkdir -p build
cp "$SRC" build/icon-1024.png

rm -rf build/icon.iconset
mkdir -p build/icon.iconset
for s in 16 32 128 256 512; do
  sips -z $s $s build/icon-1024.png --out "build/icon.iconset/icon_${s}x${s}.png" >/dev/null
  sips -z $((s * 2)) $((s * 2)) build/icon-1024.png --out "build/icon.iconset/icon_${s}x${s}@2x.png" >/dev/null
done
iconutil -c icns build/icon.iconset -o build/mygou.app/Contents/Resources/icon.icns

touch build/mygou.app
echo "✓ Successfully updated build/mygou.app icon!"
