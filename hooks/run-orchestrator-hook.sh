#!/usr/bin/env bash
set -euo pipefail

EVENT=${1:-}
case "$EVENT" in
  session-start|post-compact) ;;
  *) exit 0 ;;
esac

CLI=${ORCHESTRATOR_CLI:-}
if [[ -n "$CLI" && ! -x "$CLI" ]]; then
  CLI=""
fi
if [[ -z "$CLI" && -n "${PLUGIN_ROOT:-}" && -x "${PLUGIN_ROOT}/cli/orchestrator" ]]; then
  CLI="${PLUGIN_ROOT}/cli/orchestrator"
fi
if [[ -z "$CLI" && -n "${PLUGIN_ROOT:-}" && -x "${PLUGIN_ROOT}/bin/orchestrator" ]]; then
  CLI="${PLUGIN_ROOT}/bin/orchestrator"
fi
if [[ -z "$CLI" ]]; then
  CLI=$(command -v orchestrator 2>/dev/null || true)
fi
if [[ -z "$CLI" ]]; then
  exit 0
fi

exec "$CLI" hook "$EVENT"
