#!/usr/bin/env sh

# Tiny bootstrapper for the released Go CLI. It deliberately keeps the
# dotfiles checkout as the configuration source and only downloads the binary.
set -eu

repo='emmsixx/dotfiles'
version="${DOTFILES_VERSION:-latest}"
install_dir="${DOTFILES_BIN_DIR:-$HOME/.local/bin}"
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

case "$(uname -s)" in
    Darwin) os='Darwin'; os_lower='darwin' ;;
    Linux) os='Linux'; os_lower='linux' ;;
    *) printf 'Unsupported operating system: %s\n' "$(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64|amd64) arch='amd64' ;;
    arm64|aarch64) arch='arm64' ;;
    *) printf 'Unsupported CPU architecture: %s\n' "$(uname -m)" >&2; exit 1 ;;
esac

base="https://github.com/${repo}/releases"
if [ "$version" = latest ]; then
    release_url="$base/latest/download"
else
    release_url="$base/download/$version"
fi

tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/dotfiles.XXXXXX")
trap 'rm -rf "$tmpdir"' EXIT HUP INT TERM

printf 'Downloading dotfiles %s for %s/%s…\n' "$version" "$os" "$arch"
curl -fsSL "$release_url/checksums.txt" -o "$tmpdir/checksums.txt"

archive=''
for candidate in \
    "dotfiles_${os}_${arch}.tar.gz" \
    "dotfiles_${os_lower}_${arch}.tar.gz"; do
    if awk -v file="$candidate" '$2 == file || $2 == "*" file { found = 1 } END { exit !found }' "$tmpdir/checksums.txt"; then
        archive="$candidate"
        break
    fi
done
if [ -z "$archive" ] && [ "$arch" = amd64 ]; then
    for candidate in "dotfiles_${os}_x86_64.tar.gz" "dotfiles_${os_lower}_x86_64.tar.gz"; do
        if awk -v file="$candidate" '$2 == file || $2 == "*" file { found = 1 } END { exit !found }' "$tmpdir/checksums.txt"; then
            archive="$candidate"
            break
        fi
    done
fi
if [ -z "$archive" ]; then
    printf 'No release archive matches %s/%s.\n' "$os" "$arch" >&2
    exit 1
fi
curl -fsSL "$release_url/$archive" -o "$tmpdir/$archive"

expected=$(awk -v file="$archive" '$2 == file || $2 == "*" file { print $1; exit }' "$tmpdir/checksums.txt")
if [ -z "$expected" ]; then
    printf 'Checksum for %s was not found in the release manifest.\n' "$archive" >&2
    exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$tmpdir/$archive" | awk '{print $1}')
else
    actual=$(shasum -a 256 "$tmpdir/$archive" | awk '{print $1}')
fi
if [ "$expected" != "$actual" ]; then
    printf 'Checksum verification failed.\n' >&2
    exit 1
fi

tar -xzf "$tmpdir/$archive" -C "$tmpdir"
binary=$(find "$tmpdir" -type f -name dotfiles -perm -u+x | head -n 1)
if [ -z "$binary" ]; then
    printf 'The release archive did not contain a dotfiles binary.\n' >&2
    exit 1
fi
mkdir -p "$install_dir"
install -m 0755 "$binary" "$install_dir/dotfiles"

exec "$install_dir/dotfiles" setup --repo "$script_dir" "$@"
