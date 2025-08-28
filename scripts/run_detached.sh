#!/usr/bin/env bash
set -euo pipefail

# Run a long script detached with nohup, capture stdout/stderr to a timestamped log,
# and write the PID file alongside for monitoring/cancellation.
#
# Usage:
#   scripts/run_detached.sh <script> [args...]
#
# Example:
#   scripts/run_detached.sh scripts/run_compare_cluster.sh -i SD5000_1 -o outputs/compare

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <script> [args...]" >&2
  exit 2
fi

TARGET="$1"; shift || true
[[ -f "$TARGET" ]] || { echo "Script not found: $TARGET" >&2; exit 2; }

STAMP="$(date +%Y%m%d-%H%M%S)"
LOGDIR="logs"; mkdir -p "$LOGDIR"
BASENAME="$(basename "$TARGET" .sh)"
LOGFILE="$LOGDIR/${BASENAME}.${STAMP}.nohup.log"
PIDFILE="$LOGDIR/${BASENAME}.${STAMP}.pid"

echo "[detached] Launching: $TARGET $*"
echo "[detached] Log: $LOGFILE"
echo "[detached] PID: $PIDFILE"

nohup bash "$TARGET" "$@" > "$LOGFILE" 2>&1 &
echo $! > "$PIDFILE"
echo "[detached] Started with PID $(cat "$PIDFILE")"

