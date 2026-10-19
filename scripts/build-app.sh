#!/bin/bash
# Builds build/mygou.app: frontend bundle, Go binary, icon, Info.plist.
set -euo pipefail
cd "$(dirname "$0")/.."

(cd frontend && npm run build)

mkdir -p build
go build -mod=vendor -trimpath -ldflags "-s -w" -o build/mygou .

rm -rf build/mygou.app
mkdir -p build/mygou.app/Contents/MacOS build/mygou.app/Contents/Resources
cp build/mygou build/mygou.app/Contents/MacOS/mygou
cp packaging/Info.plist build/mygou.app/Contents/Info.plist

go run ./cmd/mkicon -out build/icon-1024.png
rm -rf build/icon.iconset
mkdir -p build/icon.iconset
for s in 16 32 128 256 512; do
  sips -z $s $s build/icon-1024.png --out "build/icon.iconset/icon_${s}x${s}.png" >/dev/null
  sips -z $((s * 2)) $((s * 2)) build/icon-1024.png --out "build/icon.iconset/icon_${s}x${s}@2x.png" >/dev/null
done
iconutil -c icns build/icon.iconset -o build/mygou.app/Contents/Resources/icon.icns

codesign --force --deep --sign - build/mygou.app
echo "built build/mygou.app"
