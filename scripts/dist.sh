#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
version=$(cat VERSION)
case "$version" in
  *[!0-9.]*|'') printf '%s\n' 'Invalid VERSION.' >&2; exit 1 ;;
esac
mkdir -p dist
# Releases are archived on GitHub; retain only the current local build.
for artifact in dist/domain_*_darwin_arm64.tar.gz dist/domain_*_darwin_amd64.tar.gz dist/checksums_*.txt; do
  [ ! -f "$artifact" ] || rm -f "$artifact"
done
for arch in arm64 amd64; do
  name="domain_${version}_darwin_${arch}"
  staging=$(mktemp -d)
  trap 'rm -rf "$staging"' EXIT HUP INT TERM
  CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" go build -trimpath -ldflags='-s -w' -o "$staging/domain" ./cmd/domain
  cp README.md CHANGELOG.md LICENSE VERSION "$staging/"
  COPYFILE_DISABLE=1 tar -czf "dist/${name}.tar.gz" -C "$staging" domain README.md CHANGELOG.md LICENSE VERSION
  rm -rf "$staging"
  trap - EXIT HUP INT TERM
done
(cd dist && shasum -a 256 "domain_${version}_darwin_arm64.tar.gz" "domain_${version}_darwin_amd64.tar.gz" > "checksums_${version}.txt")
