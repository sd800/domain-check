#!/bin/sh
# Install the latest release, or an already-built executable supplied as $1.
set -eu
# Use macOS system tools rather than executables injected through the caller's PATH.
PATH=/usr/bin:/bin:/usr/sbin:/sbin
export PATH
umask 077

work=
pending=
cleanup() {
  [ -z "$pending" ] || rm -f "$pending"
  [ -z "$work" ] || rm -rf "$work"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

download() {
  curl -q --proto '=https' --proto-redir '=https' --connect-timeout 15 \
    --max-time 180 --max-filesize 67108864 -fsSL --retry 3 -o "$1" "$2"
}

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
  download "$work/release.json" \
    https://api.github.com/repos/sd800/domain-check/releases/latest
  tag=$(/usr/bin/plutil -extract tag_name raw -o - "$work/release.json")
  if ! printf '%s\n' "$tag" | grep -Eq '^v[0-9]{2}\.[1-9][0-9]?\.[1-9][0-9]?\.[1-9][0-9]*$'; then
    printf '%s\n' 'Invalid release version.' >&2
    exit 1
  fi
  version=${tag#v}
  archive="domain_${version}_darwin_${arch}.tar.gz"
  base="https://github.com/sd800/domain-check/releases/download/$tag"
  download "$work/$archive" "$base/$archive"
  download "$work/checksums.txt" "$base/checksums_${version}.txt"

  # Verify exactly this archive before extracting or replacing the executable.
  (
    cd "$work"
    awk -v file="$archive" '$2 == file { if (NF != 2 || length($1) != 64 || $1 ~ /[^0-9a-fA-F]/) { bad=1; next } print; n++ } END { exit bad || n != 1 }' \
      checksums.txt > verify.txt
    shasum -a 256 -c verify.txt
  )
  # Accept only the expected archive layout, with one regular executable member.
  tar -tf "$work/$archive" > "$work/members.txt"
  awk '
    BEGIN {
      split("domain README.md CHANGELOG.md LICENSE VERSION SECURITY.md docs/ docs/technical.md docs/domains.md docs/security.md docs/quality-assurance.md docs/references.md", names, " ")
      for (i in names) allowed[names[i]]=1
    }
    !allowed[$0] || seen[$0]++ { bad=1 }
    END { exit bad || !seen["domain"] || !seen["README.md"] || !seen["CHANGELOG.md"] || !seen["LICENSE"] || !seen["VERSION"] }
  ' "$work/members.txt"
  tar -tvf "$work/$archive" domain > "$work/domain-entry.txt"
  awk 'NR != 1 || substr($0,1,1) != "-" { bad=1 } END { exit bad || NR != 1 }' "$work/domain-entry.txt"
  (ulimit -f 131072; tar -xOf "$work/$archive" domain > "$work/domain")
  [ -s "$work/domain" ]
  binary="$work/domain"
else
  # Source builds share installation and PATH setup without downloading a release.
  binary=$1
  case "$binary" in /*) ;; *) binary="./$binary" ;; esac
fi

# Install for this user; administrator privileges are not needed.
mkdir -p "$HOME/.local/bin"
if [ -d "$HOME/.local/bin/domain" ] && [ ! -L "$HOME/.local/bin/domain" ]; then
  printf '%s\n' 'The installation target is a directory.' >&2
  exit 1
fi
pending=$(mktemp "$HOME/.local/bin/.domain-install.XXXXXXXX")
install -m 755 "$binary" "$pending"
# Rename replaces a pre-existing symlink itself instead of writing through it.
mv -fh "$pending" "$HOME/.local/bin/domain"
pending=

# Add only a standard PATH entry, once, respecting an existing ZDOTDIR setting.
# The README also exports PATH in the calling terminal, where this script cannot.
profile="${ZDOTDIR:-$HOME}/.zshrc"
mkdir -p "$(dirname "$profile")"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$profile" 2>/dev/null ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$profile"
