#!/usr/bin/env bash
set -euo pipefail

if command -v apt-get >/dev/null 2>&1; then
  echo "[install_deps] Installing packages via apt-get"
  sudo apt-get update
  sudo apt-get install -y openjdk-17-jdk curl unzip git python3
  echo "[install_deps] Done."
else
  cat <<'EOF'
[install_deps] This helper only supports apt-based systems.
Please install the following manually if they are missing:
  - Java 11 or newer (java command on PATH)
  - curl, unzip, tar, git
  - Python 3 (for validate.sh --limit)
EOF
fi

