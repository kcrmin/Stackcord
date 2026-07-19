#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
BINARY=""
OUTPUT=""
WORKSPACE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --binary) BINARY=$2; shift 2 ;;
    --output) OUTPUT=$2; shift 2 ;;
    --workspace) WORKSPACE=$2; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

TEMP=""
if [[ -z "$BINARY" || -z "$OUTPUT" || -z "$WORKSPACE" ]]; then
  TEMP=$(mktemp -d "${TMPDIR:-/tmp}/stackcord-dogfood.XXXXXX")
fi
if [[ -z "$BINARY" ]]; then
  BINARY="$TEMP/stackcord"
  (cd "$ROOT/cli" && go build -trimpath -o "$BINARY" ./cmd/stackcord)
fi
if [[ -z "$OUTPUT" ]]; then OUTPUT="$TEMP/result.json"; fi
if [[ -z "$WORKSPACE" ]]; then WORKSPACE="$TEMP/fixture"; fi

python3 "$ROOT/dogfood/run.py" --binary "$BINARY" --output "$OUTPUT" --workspace "$WORKSPACE"
