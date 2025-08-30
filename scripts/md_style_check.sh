#!/usr/bin/env bash
set -euo pipefail

# Simple Markdown style checker based on Layth's writing guide.
# - Scans README.md and docs/*.md
# - Flags banned words/phrases and headings deeper than H3
# - Suggests language tags for code fences

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
shopt -s nullglob
files=( "$ROOT"/*.md "$ROOT"/docs/*.md )
shopt -u nullglob

if [[ ${#files[@]} -eq 0 ]]; then echo "No markdown files found"; exit 0; fi

ban=(
  'leverage' 'utilize' 'synergy' 'synergize' 'holistic' 'innovative' 'innovation'
  'disruptive' 'cutting-edge' 'state-of-the-art' 'world-class' 'best-in-class'
  'game-changing' 'revolutionary' 'paradigm' 'thought leader' 'robust' 'seamless'
  'frictionless' 'intuitive' 'user-friendly' 'empower' 'unlock' 'elevate' 'deep dive'
  'circle back' 'low-hanging fruit' 'move the needle' 'boil the ocean' 'bandwidth'
)
ban_phr=(
  "In today's fast-paced world" "rapidly evolving landscape" "It's not just"
  "This document aims to" "At the end of the day" "Needless to say" "It should be noted"
  "To summarize" "In conclusion" "We're excited" "We're thrilled" "best practices" "industry standard"
  "Next generation" "Enterprise-grade" "Military-grade"
)

fail=0
for f in "${files[@]}"; do
  rel="${f#$ROOT/}"
  # Headings deeper than H3
  if rg -n '^#{4,}\s' -S "$f" >/dev/null; then
    echo "[H] $rel: headings deeper than H3 found"; fail=1
    rg -n '^#{4,}\s' -S "$f" | sed 's/^/    /'
  fi
  # Banned words (case-insensitive)
  if [[ "$rel" != "docs/WRITING_STYLE.md" ]]; then
    for w in "${ban[@]}"; do
      if rg -n "\b$w\b" -i -S "$f" >/dev/null; then
        echo "[B] $rel: banned term '$w'"; fail=1
        rg -n "\b$w\b" -i -S "$f" | head -n 3 | sed 's/^/    /'
      fi
    done
    for p in "${ban_phr[@]}"; do
      if rg -n "$p" -S "$f" >/dev/null; then
        echo "[P] $rel: phrase '$p'"; fail=1
        rg -n "$p" -S "$f" | head -n 3 | sed 's/^/    /'
      fi
    done
  fi
  # Code fences without language tag
  if rg -n '^```$' -S "$f" >/dev/null; then
    echo "[C] $rel: code fence without language tag";
    rg -n '^```$' -S "$f" | sed 's/^/    /'
  fi
done

if (( fail )); then
  if [[ "${STRICT:-0}" == "1" ]]; then
    echo "\nStyle check FAILED (STRICT=1). Fix issues above." >&2
    exit 1
  else
    echo "\nStyle check WARN (set STRICT=1 to fail)." >&2
  fi
else
  echo "Style check OK (${#files[@]} files)"
fi
