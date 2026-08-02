#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="$root/dist"
if [[ -n "$(git -C "$root" status --porcelain --untracked-files=all)" ]]; then
  echo "release builds require a clean worktree" >&2
  exit 2
fi
version="${MYFERENCE_VERSION:-$(git -C "$root" describe --always)}"
commit="$(git -C "$root" rev-parse HEAD)"
export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/myference-release-go-cache}"
stage="$(mktemp -d "${TMPDIR:-/tmp}/myference-release.XXXXXX")"
trap 'rm -rf "$stage"' EXIT

mkdir -p "$dist"
printf 'version=%s\ncommit=%s\n' "$version" "$commit" > "$stage/VERSION"

build_cli() {
  local goos="$1" goarch="$2" output="$3"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$version -X main.commit=$commit" -o "$output" "$root/cli/cmd/myference"
}

mkdir -p "$stage/windows-amd64" "$stage/macos-amd64" "$stage/macos-arm64" "$stage/server-linux-amd64"
build_cli windows amd64 "$stage/windows-amd64/myference.exe"
build_cli darwin amd64 "$stage/macos-amd64/myference"
build_cli darwin arm64 "$stage/macos-arm64/myference"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$stage/windows-amd64/myference-agent-proxy" "$root/cli/cmd/myference-agent-proxy"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$stage/macos-amd64/myference-agent-proxy" "$root/cli/cmd/myference-agent-proxy"
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$stage/macos-arm64/myference-agent-proxy" "$root/cli/cmd/myference-agent-proxy"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$stage/server-linux-amd64/myference-server" "$root/server/cmd/myference-server"
cp "$root/packaging/windows/install.ps1" "$stage/windows-amd64/install.ps1"
cp "$root/packaging/macos/com.myference.provider.plist" "$stage/macos-amd64/com.myference.provider.plist"
cp "$root/packaging/macos/com.myference.provider.plist" "$stage/macos-arm64/com.myference.provider.plist"
for directory in "$stage"/*-amd64 "$stage"/macos-arm64; do cp "$stage/VERSION" "$directory/VERSION"; done

for artifact in "$stage"/*/myference*; do
  if strings "$artifact" | grep -Eq 'ac0974bec39a17e36ba4a6b4d238ff944|mach-1\.machine-secret'; then
    echo "secret-like test material found in $artifact" >&2
    exit 1
  fi
done

find "$stage" -type f -exec touch -t 198001010000 {} +
for package in windows-amd64 macos-amd64 macos-arm64 server-linux-amd64; do
  (cd "$stage/$package" && zip -X -q -r "$dist/myference-$package-$version.zip" .)
done
(cd "$dist" && shasum -a 256 myference-*"$version".zip > SHA256SUMS)

echo "release artifacts written to $dist"
