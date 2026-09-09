#!/usr/bin/env bash
set -euo pipefail

EVENT=${1:-}
case "$EVENT" in
  session-start|post-compact) ;;
  *) exit 0 ;;
esac

ROOT=${PLUGIN_ROOT:-${CLAUDE_PLUGIN_ROOT:-}}
CLI=${STACKCORD_CLI:-}
if [[ -n "$CLI" && ! -x "$CLI" ]]; then
  CLI=""
fi
if [[ -z "$CLI" && -n "${ROOT:-}" && -x "${ROOT}/cli/stackcord" ]]; then
  CLI="${ROOT}/cli/stackcord"
fi
if [[ -z "$CLI" && -n "${ROOT:-}" && -x "${ROOT}/bin/stackcord" ]]; then
  CLI="${ROOT}/bin/stackcord"
fi
if [[ -z "$CLI" && -n "$ROOT" && -f "${ROOT}/bin/stackcord.exe" ]]; then
  CLI="${ROOT}/bin/stackcord.exe"
fi
if [[ -z "$CLI" ]]; then
  CLI=$(command -v stackcord 2>/dev/null || true)
fi
if [[ -z "$CLI" ]]; then
  exit 0
fi

exec "$CLI" hook "$EVENT"
