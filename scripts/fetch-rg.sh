#!/usr/bin/env bash
# Download microsoft/ripgrep-prebuilt for GOOS/GOARCH into the Go embed dir.
set -euo pipefail

VERSION=v15.0.1
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST_DIR="$ROOT/services/internal/rgbin/bin"
DEST="$DEST_DIR/rg.bin"
STAMP="$ROOT/services/internal/rgbin/rg.stamp"
GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"

asset_for() {
  case "$1/$2" in
    windows/amd64) echo "ripgrep-${VERSION}-x86_64-pc-windows-msvc.zip" ;;
    windows/arm64) echo "ripgrep-${VERSION}-aarch64-pc-windows-msvc.zip" ;;
    linux/amd64) echo "ripgrep-${VERSION}-x86_64-unknown-linux-musl.tar.gz" ;;
    linux/arm64) echo "ripgrep-${VERSION}-aarch64-unknown-linux-musl.tar.gz" ;;
    darwin/amd64) echo "ripgrep-${VERSION}-x86_64-apple-darwin.tar.gz" ;;
    darwin/arm64) echo "ripgrep-${VERSION}-aarch64-apple-darwin.tar.gz" ;;
    *) echo "unsupported GOOS/GOARCH: $1/$2" >&2; return 1 ;;
  esac
}

sha_for() {
  case "$1" in
    ripgrep-${VERSION}-x86_64-pc-windows-msvc.zip)
      echo bd28761f4918ea8fcb7a95f636b4422a915d55af268d9805be82d8ce0fdfc823 ;;
    ripgrep-${VERSION}-aarch64-pc-windows-msvc.zip)
      echo cc36bae403f25c838d25a3c65ba64f38cc00904652e89d6377b5ceaf66df8432 ;;
    ripgrep-${VERSION}-x86_64-unknown-linux-musl.tar.gz)
      echo 4499958bfd5252df3d9e7504127fd448e4a14fbf2805ef4f14baaa1bcf775188 ;;
    ripgrep-${VERSION}-aarch64-unknown-linux-musl.tar.gz)
      echo dd3738a4b6e8df0fb3bc3edc5af352c4c39e0d97ad118a23e5176bdc5d48ba08 ;;
    ripgrep-${VERSION}-x86_64-apple-darwin.tar.gz)
      echo 591c693e80bb444ef1907b2a906feb9c77bcafe1cdf509107cc75dcf0e875bd2 ;;
    ripgrep-${VERSION}-aarch64-apple-darwin.tar.gz)
      echo 2fa16464fd8638588a67c7fc172d3c4b57fbdc65dff366e10b0b0e90734628a6 ;;
    *) echo "missing sha for $1" >&2; return 1 ;;
  esac
}

check_sha() {
  local want="$1" file="$2"
  if command -v sha256sum >/dev/null 2>&1; then
    echo "${want}  ${file}" | sha256sum -c -
  else
    echo "${want}  ${file}" | shasum -a 256 -c -
  fi
}

want="${VERSION}-${GOOS}-${GOARCH}"
if [[ -f "$DEST" && -f "$STAMP" && "$(cat "$STAMP")" == "$want" ]]; then
  echo "rg.bin already $want"
  exit 0
fi

asset="$(asset_for "$GOOS" "$GOARCH")"
sha="$(sha_for "$asset")"
url="https://github.com/microsoft/ripgrep-prebuilt/releases/download/${VERSION}/${asset}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
echo "fetching $asset"
curl -fsSL -o "$tmp/asset" "$url"
check_sha "$sha" "$tmp/asset"

mkdir -p "$tmp/out"
case "$asset" in
  *.zip) unzip -o -q -d "$tmp/out" "$tmp/asset" ;;
  *.tar.gz) tar -xzf "$tmp/asset" -C "$tmp/out" ;;
  *) echo "unknown archive $asset" >&2; exit 1 ;;
esac

bin="$(find "$tmp/out" -type f \( -name rg -o -name rg.exe \) | head -n 1)"
if [[ -z "$bin" ]]; then
  echo "rg binary not found in $asset" >&2
  exit 1
fi

mkdir -p "$DEST_DIR"
cp "$bin" "$DEST"
echo "$want" > "$STAMP"
echo "wrote $DEST ($want)"
