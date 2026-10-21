#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
prefix=${PREFIX:-"$HOME/.local"}
staging=${DESTDIR:-}
mode=${1:-install}
case "$mode" in
  install|--shell-only|--uninstall) ;;
  *) printf '%s\n' 'Usage: sh install.sh [--shell-only|--uninstall]' >&2; exit 2 ;;
esac

shell_file="$staging$prefix/share/domain-check/domain.zsh"
if [ "$mode" = --uninstall ]; then
  rm -f "$staging$prefix/bin/domain" "$shell_file"
else
  if [ "$mode" != --shell-only ]; then
    binary="$root/domain"
    [ -f "$binary" ] || binary="$root/bin/domain"
    install -d "$staging$prefix/bin"
    install -m 755 "$binary" "$staging$prefix/bin/domain"
  fi
  install -d "$(dirname -- "$shell_file")"
  install -m 644 "$root/domain.zsh" "$shell_file"
fi

# A staged package must never change the calling user's shell configuration.
[ -z "$staging" ] || exit 0
integration=${DOMAIN_ZSH_INTEGRATION:-auto}
login_shell=${SHELL:-}
case "$integration" in
  0) exit 0 ;;
  auto) [ "${login_shell##*/}" = zsh ] || exit 0 ;;
  1) ;;
  *) printf '%s\n' 'DOMAIN_ZSH_INTEGRATION must be auto, 0, or 1.' >&2; exit 2 ;;
esac

rc="${ZDOTDIR:-$HOME}/.zshrc"
# Follow existing dotfile symlinks without replacing them or keeping backups.
links=0
while [ -L "$rc" ]; do
  links=$((links + 1))
  [ "$links" -le 32 ] || { printf '%s\n' 'Cannot resolve .zshrc symlink.' >&2; exit 1; }
  target=$(readlink "$rc")
  case "$target" in
    /*) rc=$target ;;
    *) rc="$(dirname -- "$rc")/$target" ;;
  esac
done
if [ "$mode" = --uninstall ] && [ ! -e "$rc" ]; then
  exit 0
fi
mkdir -p "$(dirname -- "$rc")"
temporary=$(mktemp "$(dirname -- "$rc")/.domain-zsh.XXXXXX")
trap 'rm -f "$temporary"' EXIT HUP INT TERM
if [ -f "$rc" ]; then
  cp -p "$rc" "$temporary"
  if ! awk '
    $0 == "# >>> Domain Check from Registries >>>" { if (inside || seen++) exit 1; inside=1; next }
    $0 == "# <<< Domain Check from Registries <<<" { if (!inside) exit 1; inside=0; next }
    !inside { print }
    END { if (inside) exit 1 }
  ' "$rc" > "$temporary"; then
    printf '%s\n' 'Cannot update the managed zsh block; existing .zshrc was preserved.' >&2
    exit 1
  fi
elif [ -e "$rc" ]; then
  printf '%s\n' 'The zsh configuration path is not a regular file.' >&2
  exit 1
fi
if [ "$mode" != --uninstall ]; then
  # Single-quote the source path, including prefixes containing apostrophes.
  quoted=$(printf '%s' "$prefix/share/domain-check/domain.zsh" | sed "s/'/'\\\\''/g")
  printf '%s\n%s\n%s\n' \
    '# >>> Domain Check from Registries >>>' \
    "[[ ! -r '$quoted' ]] || source '$quoted'" \
    '# <<< Domain Check from Registries <<<' >> "$temporary"
fi
if [ ! -f "$rc" ] || ! cmp -s "$temporary" "$rc"; then
  mv -f "$temporary" "$rc"
fi
printf '%s\n' 'zsh integration updated. Open a new terminal or reload your .zshrc.'
