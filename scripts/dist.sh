#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
version=$(cat VERSION)
case "$version" in
  *[!0-9.]*|'') printf '%s\n' 'Invalid VERSION.' >&2; exit 1 ;;
esac
mkdir -p build/dist
# Releases are archived on GitHub; retain only the current local build.
for artifact in build/dist/domain-check_*_darwin_arm64.tar.gz build/dist/domain-check_*_darwin_amd64.tar.gz build/dist/domain_*_darwin_arm64.tar.gz build/dist/domain_*_darwin_amd64.tar.gz build/dist/checksums_*.txt; do
  [ ! -f "$artifact" ] || rm -f "$artifact"
done
for arch in arm64 amd64; do
  name="domain-check_${version}_darwin_${arch}"
  staging=$(mktemp -d)
  trap 'rm -rf "$staging"' EXIT HUP INT TERM
  CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" go build -trimpath -ldflags='-s -w' -o "$staging/domain-check" ./cmd/domain
  /usr/bin/codesign --force --sign - --identifier localhost.domain-check "$staging/domain-check"
  cp README.md CHANGELOG.md LICENSE VERSION "$staging/"
  mkdir "$staging/docs"
  cp docs/technical.md docs/domains.md docs/security.md docs/quality-assurance.md docs/references.md "$staging/docs/"
  COPYFILE_DISABLE=1 tar -czf "build/dist/${name}.tar.gz" -C "$staging" domain-check README.md CHANGELOG.md LICENSE VERSION docs/
  rm -rf "$staging"
  trap - EXIT HUP INT TERM
done
(cd build/dist && shasum -a 256 "domain-check_${version}_darwin_arm64.tar.gz" "domain-check_${version}_darwin_amd64.tar.gz" > "checksums_${version}.txt")
