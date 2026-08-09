#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
installer="$root/web/public/install.sh"
windows_installer="$root/web/public/install.ps1"
service_installer="$root/packaging/windows/install.ps1"

sh -n "$installer"
test -f "$windows_installer"
grep -F 'Get-FileHash' "$windows_installer" >/dev/null
grep -F 'SetEnvironmentVariable' "$windows_installer" >/dev/null
if grep -F 'Headless' "$service_installer" >/dev/null; then
  echo 'Windows service installer still exposes headless mode' >&2
  exit 1
fi
if grep -F 'windows headless' "$service_installer" >/dev/null; then
  echo 'Windows service installer still invokes the removed command' >&2
  exit 1
fi

fixture=$(mktemp -d "${TMPDIR:-/tmp}/myference-installer-test.XXXXXX")
trap 'rm -rf "$fixture"' EXIT HUP INT TERM
tag=v9.8.7-test
release="$fixture/releases/$tag"
mkdir -p "$release" "$fixture/stage"

for arch in amd64 arm64; do
  package="myference-macos-$arch-$tag"
  package_dir="$fixture/stage/$package"
  mkdir -p "$package_dir"
  printf '%s\n' "cli-$arch" > "$package_dir/myference"
  printf '%s\n' "proxy-$arch" > "$package_dir/myference-agent-proxy"
  chmod +x "$package_dir/myference" "$package_dir/myference-agent-proxy"
  (cd "$package_dir" && zip -X -q "$release/$package.zip" myference myference-agent-proxy)
done
(cd "$release" && shasum -a 256 myference-*.zip > SHA256SUMS)

if MYFERENCE_RELEASE_TAG="$tag" \
  MYFERENCE_DOWNLOAD_BASE="file://$fixture/releases" \
  MYFERENCE_INSTALL_DIR="$fixture/linux" \
  MYFERENCE_OS=Linux \
  MYFERENCE_ARCH=amd64 \
  sh "$installer" >/dev/null 2>&1; then
  echo 'macOS installer accepted a non-macOS host' >&2
  exit 1
fi
if MYFERENCE_OS=Darwin MYFERENCE_ARCH=sparc sh "$installer" >/dev/null 2>&1; then
  echo 'macOS installer accepted an unsupported architecture' >&2
  exit 1
fi

for arch in amd64 arm64; do
  install_dir="$fixture/install-$arch"
  MYFERENCE_RELEASE_TAG="$tag" \
    MYFERENCE_DOWNLOAD_BASE="file://$fixture/releases" \
    MYFERENCE_INSTALL_DIR="$install_dir" \
    MYFERENCE_OS=Darwin \
    MYFERENCE_ARCH="$arch" \
    sh "$installer" >/dev/null
  test "$(cat "$install_dir/myference")" = "cli-$arch"
  test "$(cat "$install_dir/myference-agent-proxy")" = "proxy-$arch"
  test -x "$install_dir/myference"
  test -x "$install_dir/myference-agent-proxy"
done

rollback_dir="$fixture/rollback"
fake_bin="$fixture/fake-bin"
mkdir -p "$rollback_dir" "$fake_bin"
printf '%s\n' '#!/bin/sh' 'case "$1" in */.myference.new-*) exit 55 ;; esac' 'exec /bin/mv "$@"' > "$fake_bin/mv"
chmod +x "$fake_bin/mv"
printf 'old-cli\n' > "$rollback_dir/myference"
printf 'old-proxy\n' > "$rollback_dir/myference-agent-proxy"
if PATH="$fake_bin:$PATH" \
  MYFERENCE_RELEASE_TAG="$tag" \
  MYFERENCE_DOWNLOAD_BASE="file://$fixture/releases" \
  MYFERENCE_INSTALL_DIR="$rollback_dir" \
  MYFERENCE_OS=Darwin \
  MYFERENCE_ARCH=amd64 \
  sh "$installer" >/dev/null 2>&1; then
  echo 'installer did not surface a destination replacement failure' >&2
  exit 1
fi
test "$(cat "$rollback_dir/myference")" = 'old-cli'
test "$(cat "$rollback_dir/myference-agent-proxy")" = 'old-proxy'

signal_dir="$fixture/signal-rollback"
signal_bin="$fixture/signal-bin"
mkdir -p "$signal_dir" "$signal_bin"
printf '%s\n' '#!/bin/sh' 'case "$1" in */.myference-agent-proxy.new-*) kill -TERM "$PPID"; sleep 1; exit 143 ;; esac' 'exec /bin/mv "$@"' > "$signal_bin/mv"
chmod +x "$signal_bin/mv"
printf 'old-cli\n' > "$signal_dir/myference"
printf 'old-proxy\n' > "$signal_dir/myference-agent-proxy"
if PATH="$signal_bin:$PATH" \
  MYFERENCE_RELEASE_TAG="$tag" \
  MYFERENCE_DOWNLOAD_BASE="file://$fixture/releases" \
  MYFERENCE_INSTALL_DIR="$signal_dir" \
  MYFERENCE_OS=Darwin \
  MYFERENCE_ARCH=amd64 \
  sh "$installer" >/dev/null 2>&1; then
  echo 'installer ignored an interruption during replacement' >&2
  exit 1
fi
test "$(cat "$signal_dir/myference")" = 'old-cli'
test "$(cat "$signal_dir/myference-agent-proxy")" = 'old-proxy'

printf 'tampered\n' >> "$release/myference-macos-arm64-$tag.zip"
if MYFERENCE_RELEASE_TAG="$tag" \
  MYFERENCE_DOWNLOAD_BASE="file://$fixture/releases" \
  MYFERENCE_INSTALL_DIR="$fixture/rejected" \
  MYFERENCE_OS=Darwin \
  MYFERENCE_ARCH=arm64 \
  sh "$installer" >/dev/null 2>&1; then
  echo 'installer accepted a tampered archive' >&2
  exit 1
fi
test ! -e "$fixture/rejected/myference"

echo 'installer tests passed'
