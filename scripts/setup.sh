#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"

if [[ "${1:-}" == "--deps" ]]; then
  bash "${BASE_DIR}/scripts/install_deps.sh"
fi

bash "${BASE_DIR}/scripts/get_bundle.sh" "${2:-bundle}"

cat <<'EOF'
[setup] Bundle ready. Next steps:
  1. bash scripts/flight_check.sh
  2. bash scripts/validate_mimic.sh (optional smoke test)
  3. bash scripts/run_pipeline.sh --pipeline sectioned -i /path/to/notes -o outputs/run1
EOF
