#!/bin/sh
set -eu

repository=kunalshah017/myference
operating_system=${MYFERENCE_OS:-$(uname -s)}
if [ "$operating_system" != Darwin ]; then
  echo "This installer supports macOS only (detected $operating_system)" >&2
  exit 1
fi
machine=${MYFERENCE_ARCH:-$(uname -m)}
case "$machine" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "Unsupported macOS architecture: $machine" >&2; exit 1 ;;
esac

tag=${MYFERENCE_RELEASE_TAG:-}
if [ -z "$tag" ]; then
  latest=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$repository/releases/latest")
  tag=${latest##*/}
fi
case "$tag" in
  ''|*[!A-Za-z0-9._-]*) echo "Invalid release tag: $tag" >&2; exit 1 ;;
esac

asset="myference-macos-$arch-$tag.zip"
base=${MYFERENCE_DOWNLOAD_BASE:-"https://github.com/$repository/releases/download"}
install_dir=${MYFERENCE_INSTALL_DIR:-/usr/local/bin}
use_sudo=0
if [ -z "${MYFERENCE_INSTALL_DIR:-}" ] && { [ ! -d "$install_dir" ] || [ ! -w "$install_dir" ]; }; then
  use_sudo=1
fi
run_admin() {
  if [ "$use_sudo" -eq 1 ]; then sudo "$@"; else "$@"; fi
}
temporary=$(mktemp -d "${TMPDIR:-/tmp}/myference-install.XXXXXX")
proxy_new=
cli_new=
transaction_started=0
cleanup() {
  if [ "$transaction_started" -eq 1 ] && command -v rollback_install >/dev/null 2>&1; then
    rollback_install || true
    transaction_started=0
  fi
  rm -rf "$temporary"
  [ -z "$proxy_new" ] || run_admin rm -f "$proxy_new"
  [ -z "$cli_new" ] || run_admin rm -f "$cli_new"
}
trap cleanup EXIT
trap 'cleanup; exit 129' HUP
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM

curl -fsSL "$base/$tag/$asset" -o "$temporary/$asset"
curl -fsSL "$base/$tag/SHA256SUMS" -o "$temporary/SHA256SUMS"
expected=$(awk -v name="$asset" '$2 == name || $2 == "*" name { print $1 }' "$temporary/SHA256SUMS")
if [ -z "$expected" ]; then
  echo "No checksum published for $asset" >&2
  exit 1
fi
actual=$(shasum -a 256 "$temporary/$asset" | awk '{print $1}')
if [ "$actual" != "$expected" ]; then
  echo "Checksum verification failed for $asset" >&2
  exit 1
fi

mkdir -p "$temporary/package"
unzip -q "$temporary/$asset" -d "$temporary/package"
for binary in myference myference-agent-proxy; do
  if [ ! -f "$temporary/package/$binary" ]; then
    echo "Release is missing $binary" >&2
    exit 1
  fi
done

run_admin mkdir -p "$install_dir"
transaction=$$
proxy="$install_dir/myference-agent-proxy"
cli="$install_dir/myference"
proxy_new="$install_dir/.myference-agent-proxy.new-$transaction"
cli_new="$install_dir/.myference.new-$transaction"
proxy_backup="$install_dir/.myference-agent-proxy.backup-$transaction"
cli_backup="$install_dir/.myference.backup-$transaction"
had_proxy=0
had_cli=0
installed_proxy=0
installed_cli=0

install_pair() {
  run_admin install -m 0755 "$temporary/package/myference-agent-proxy" "$proxy_new" || return 1
  run_admin install -m 0755 "$temporary/package/myference" "$cli_new" || return 1
  transaction_started=1
  if run_admin test -e "$proxy"; then run_admin mv "$proxy" "$proxy_backup" || return 1; had_proxy=1; fi
  if run_admin test -e "$cli"; then run_admin mv "$cli" "$cli_backup" || return 1; had_cli=1; fi
  installed_proxy=1
  run_admin mv "$proxy_new" "$proxy" || return 1
  installed_cli=1
  run_admin mv "$cli_new" "$cli" || return 1
}
rollback_install() {
  [ "$installed_cli" -eq 0 ] || run_admin rm -f "$cli"
  [ "$installed_proxy" -eq 0 ] || run_admin rm -f "$proxy"
  if run_admin test -e "$cli_backup"; then run_admin mv "$cli_backup" "$cli"; fi
  if run_admin test -e "$proxy_backup"; then run_admin mv "$proxy_backup" "$proxy"; fi
}
if ! install_pair; then
  rollback_install
  transaction_started=0
  echo "Installation failed; the previous Myference installation was restored" >&2
  exit 1
fi
[ "$had_cli" -eq 0 ] || run_admin rm -f "$cli_backup"
[ "$had_proxy" -eq 0 ] || run_admin rm -f "$proxy_backup"
transaction_started=0
proxy_new=
cli_new=

echo "Installed Myference $tag for macOS $arch in $install_dir"
echo "Run: myference host"
