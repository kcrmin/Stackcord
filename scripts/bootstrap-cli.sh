#!/usr/bin/env bash
set -euo pipefail

BASE_URL=""
VERSION=""
INSTALL_DIR="${HOME}/.local/bin"
OS_NAME=""
ARCH=""
TEMP_DIR=""
STAGED=""

cleanup() {
  if [[ -n "$STAGED" ]]; then rm -f "$STAGED"; fi
  if [[ -n "$TEMP_DIR" ]]; then rm -rf "$TEMP_DIR"; fi
}
trap cleanup EXIT INT TERM

usage() {
  printf 'Usage: %s --base-url URL --version VERSION [--install-dir DIR] [--os darwin] [--arch amd64|arm64]\n' "$0"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base-url) BASE_URL=${2:-}; shift 2 ;;
    --version) VERSION=${2:-}; shift 2 ;;
    --install-dir) INSTALL_DIR=${2:-}; shift 2 ;;
    --os) OS_NAME=${2:-}; shift 2 ;;
    --arch) ARCH=${2:-}; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) printf 'Unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "$BASE_URL" || -z "$VERSION" ]]; then
  usage >&2
  exit 2
fi
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  printf 'Invalid version: %s\n' "$VERSION" >&2
  exit 2
fi
case "$BASE_URL" in
  https://*) ;;
  http://127.0.0.1:*|http://localhost:*) ;;
  *) printf 'Base URL must use HTTPS (localhost HTTP is allowed for tests).\n' >&2; exit 2 ;;
esac
BASE_URL=${BASE_URL%/}

if [[ -z "$OS_NAME" ]]; then
  case "$(uname -s)" in
    Darwin) OS_NAME=darwin ;;
    *) printf 'This bootstrap supports macOS; use bootstrap-cli.ps1 on Windows.\n' >&2; exit 2 ;;
  esac
fi
if [[ -z "$ARCH" ]]; then
  case "$(uname -m)" in
    x86_64|amd64) ARCH=amd64 ;;
    arm64|aarch64) ARCH=arm64 ;;
    *) printf 'Unsupported architecture.\n' >&2; exit 2 ;;
  esac
fi
if [[ "$OS_NAME" != darwin || ( "$ARCH" != amd64 && "$ARCH" != arm64 ) ]]; then
  printf 'Unsupported platform: %s/%s\n' "$OS_NAME" "$ARCH" >&2
  exit 2
fi

ASSET="orchestrator_${OS_NAME}_${ARCH}"
RELEASE_URL="${BASE_URL}/v${VERSION}"
TEMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/orchestrator-bootstrap.XXXXXX")
CHECKSUMS="${TEMP_DIR}/checksums.txt"
DOWNLOAD="${TEMP_DIR}/${ASSET}"

command -v curl >/dev/null 2>&1 || { printf 'curl is required.\n' >&2; exit 3; }
curl --proto '=https,http' --proto-redir '=https,http' --tlsv1.2 -fsSL "${RELEASE_URL}/checksums.txt" -o "$CHECKSUMS"
curl --proto '=https,http' --proto-redir '=https,http' --tlsv1.2 -fsSL "${RELEASE_URL}/${ASSET}" -o "$DOWNLOAD"

EXPECTED=$(awk -v asset="$ASSET" '
  ($2 == asset || $2 == "*" asset) && length($1) == 64 && $1 !~ /[^0-9A-Fa-f]/ { print tolower($1) }
' "$CHECKSUMS")
if [[ ${#EXPECTED} -ne 64 || "$EXPECTED" == *$'\n'* ]]; then
  printf 'Checksum manifest must contain exactly one SHA-256 for %s.\n' "$ASSET" >&2
  exit 4
fi
if command -v shasum >/dev/null 2>&1; then
  ACTUAL=$(shasum -a 256 "$DOWNLOAD" | awk '{print tolower($1)}')
elif command -v sha256sum >/dev/null 2>&1; then
  ACTUAL=$(sha256sum "$DOWNLOAD" | awk '{print tolower($1)}')
else
  printf 'shasum or sha256sum is required.\n' >&2
  exit 3
fi
if [[ "$ACTUAL" != "$EXPECTED" ]]; then
  printf 'SHA-256 mismatch for %s.\n' "$ASSET" >&2
  exit 4
fi

chmod 0755 "$DOWNLOAD"
"$DOWNLOAD" doctor --json >/dev/null
mkdir -p "$INSTALL_DIR"
TARGET="${INSTALL_DIR}/orchestrator"
STAGED="${INSTALL_DIR}/.orchestrator.tmp.$$"
cp "$DOWNLOAD" "$STAGED"
chmod 0755 "$STAGED"
mv -f "$STAGED" "$TARGET"
STAGED=""
"$TARGET" doctor --json
printf 'Installed verified orchestrator %s at %s\n' "$VERSION" "$TARGET"
