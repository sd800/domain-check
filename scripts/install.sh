#!/bin/sh
# Install the latest release, or an already-built executable supplied as $1.
set -eu

if [ "$(uname -s)" != Darwin ]; then
  printf '%s\n' 'This installer requires macOS.' >&2
  exit 1
fi
if [ "$#" -gt 1 ]; then
  printf '%s\n' 'Usage: sh scripts/install.sh [path/to/domain]' >&2
  exit 1
fi

if [ "$#" -eq 0 ]; then
  # Match the release architecture to the Mac running this installer.
  case "$(uname -m)" in
    arm64) arch=arm64 ;;
    x86_64) arch=amd64 ;;
    *) printf '%s\n' 'Unsupported Mac architecture.' >&2; exit 1 ;;
  esac

  # Keep downloads temporary, and clean up on success, failure, or interruption.
  work=$(mktemp -d)
  trap 'rm -rf "$work"' EXIT HUP INT TERM
  curl -fsSL --retry 3 -o "$work/release.json" \
    https://api.github.com/repos/sd800/domain-check/releases/latest
  tag=$(/usr/bin/plutil -extract tag_name raw -o - "$work/release.json")
  version=${tag#v}
  case "$version" in
    ''|*[!0-9.]*) printf '%s\n' 'Invalid release version.' >&2; exit 1 ;;
  esac
  archive="domain_${version}_darwin_${arch}.tar.gz"
  base="https://github.com/sd800/domain-check/releases/download/$tag"
  curl -fL --retry 3 -o "$work/$archive" "$base/$archive"
  curl -fsSL --retry 3 -o "$work/checksums.txt" "$base/checksums_${version}.txt"

  # Verify exactly this archive before extracting or replacing the executable.
  (
    cd "$work"
    awk -v file="$archive" '$2 == file { print; n++ } END { exit n != 1 }' \
      checksums.txt > verify.txt
    shasum -a 256 -c verify.txt
  )
  tar -xOf "$work/$archive" domain > "$work/domain"
  binary="$work/domain"
else
  # Source builds share installation and PATH setup without downloading a release.
  binary=$1
fi

# Install for this user; administrator privileges are not needed.
mkdir -p "$HOME/.local/bin"
install -m 755 "$binary" "$HOME/.local/bin/domain"

# Add only a standard PATH entry, once, respecting an existing ZDOTDIR setting.
# The README also exports PATH in the calling terminal, where this script cannot.
profile="${ZDOTDIR:-$HOME}/.zshrc"
mkdir -p "$(dirname "$profile")"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$profile" 2>/dev/null ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$profile"
