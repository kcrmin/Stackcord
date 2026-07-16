#!/usr/bin/env sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
python3 "$ROOT/scripts/validate_plugin_test.py"
python3 "$ROOT/scripts/validate_plugin.py" "$ROOT"
