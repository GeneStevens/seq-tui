#!/bin/sh
# release-mac.sh builds seq-tui for macOS (Apple silicon and Intel) and packs one
# .tar.gz per architecture in dist/release, with a README line inside and a
# SHA256SUMS file beside them. It publishes nothing: README.md ("Play on
# play.spielengine.com") gives the `gh release create` command to run by hand.
#
#   scripts/release-mac.sh [VERSION]     (default: the short commit)
set -eu
cd "$(dirname "$0")/.."

version="${1:-$(git rev-parse --short HEAD 2>/dev/null || echo dev)}"
out="dist/release"
rm -rf "$out"
mkdir -p "$out"

for arch in arm64 amd64; do
	name="seq-tui-$version-darwin-$arch"
	stage="$out/$name"
	mkdir -p "$stage"
	GOOS=darwin GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o "$stage/seq-tui" ./cmd/seq-tui
	cat >"$stage/README.txt" <<TXT
seq-tui $version for macOS ($arch).
First launch: the binary is not signed. Run  xattr -d com.apple.quarantine ./seq-tui  once (or right-click it in Finder, choose Open).
Play:  SEQ_SESSION_TOKEN=<your invite code> ./seq-tui -url wss://play.spielengine.com/seq/v1 -new
TXT
	tar -C "$out" -czf "$out/$name.tar.gz" "$name"
	rm -rf "$stage"
done

(cd "$out" && shasum -a 256 ./*.tar.gz | sed 's| \./| |' >SHA256SUMS)
echo "release-mac: $version"
cat "$out/SHA256SUMS"
