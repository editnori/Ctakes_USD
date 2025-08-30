#!/usr/bin/env bash
# DEPRECATED: use scripts/build_xlsx_report.sh instead.
set -euo pipefail
BASE_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
exec bash "$BASE_DIR/scripts/build_xlsx_report.sh" "$@"
