#!/usr/bin/env bash
set -euo pipefail

EVENT=${1:-}
case "$EVENT" in
  session-start|post-compact) ;;
  *) exit 0 ;;
esac

CLI=${STACKCORD_CLI:-}
if [[ -n "$CLI" && ! -x "$CLI" ]]; then
  CLI=""
fi
if [[ -z "$CLI" && -n "${PLUGIN_ROOT:-}" && -x "${PLUGIN_ROOT}/cli/stackcord" ]]; then
  CLI="${PLUGIN_ROOT}/cli/stackcord"
fi
if [[ -z "$CLI" && -n "${PLUGIN_ROOT:-}" && -x "${PLUGIN_ROOT}/bin/stackcord" ]]; then
  CLI="${PLUGIN_ROOT}/bin/stackcord"
fi
if [[ -z "$CLI" ]]; then
  CLI=$(command -v stackcord 2>/dev/null || true)
fi
if [[ -z "$CLI" ]]; then
  exit 0
fi

exec "$CLI" hook "$EVENT"
