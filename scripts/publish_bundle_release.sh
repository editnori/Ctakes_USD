#!/usr/bin/env bash
set -euo pipefail

# Publish the cTAKES bundle (apache-ctakes-6.0.0-bin) to a GitHub Release.
#
# It will:
# 1) Build the bundle via scripts/make_bundle.sh (unless --no-build).
# 2) Create or update a release tagged "bundle" (customizable via -t).
# 3) Upload CtakesBun-bundle.tgz as a release asset (clobbers if present).
#
# Requirements:
# - GitHub CLI (gh) authenticated to your GitHub account/org.
#   Install: https://cli.github.com/ and run `gh auth login` once.
#
# Usage:
#   scripts/publish_bundle_release.sh [-t <tag>] [--no-build] [-f <bundle.tgz>]
#
# Examples:
#   scripts/publish_bundle_release.sh                 # build + upload to tag "bundle"
#   scripts/publish_bundle_release.sh -t bundle-v1    # custom tag
#   scripts/publish_bundle_release.sh --no-build -f /path/CtakesBun-bundle.tgz

TAG="bundle"
BUILD=1
BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BUNDLE_FILE="$BASE_DIR/CtakesBun-bundle.tgz"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -t|--tag) TAG="$2"; shift 2 ;;
    -f|--file) BUNDLE_FILE="$2"; shift 2 ;;
    --no-build) BUILD=0; shift ;;
    *) echo "Unknown arg: $1" >&2; exit 2 ;;
  esac
done

command -v gh >/dev/null 2>&1 || {
  echo "GitHub CLI 'gh' not found. Install from https://cli.github.com/ and run 'gh auth login'." >&2
  exit 2
}

if [[ $BUILD -eq 1 ]]; then
  echo "Building bundle via scripts/make_bundle.sh ..."
  "$BASE_DIR/scripts/make_bundle.sh"
fi

[[ -f "$BUNDLE_FILE" ]] || { echo "Bundle file not found: $BUNDLE_FILE" >&2; exit 2; }

SHA256=$(sha256sum "$BUNDLE_FILE" | awk '{print $1}')
echo "Bundle: $BUNDLE_FILE"
echo "SHA256: $SHA256"

if gh release view "$TAG" >/dev/null 2>&1; then
  echo "Release '$TAG' exists — uploading asset (clobber)."
  gh release upload "$TAG" "$BUNDLE_FILE" --clobber
else
  echo "Creating release '$TAG' and uploading asset."
  gh release create "$TAG" "$BUNDLE_FILE" \
    --title "cTAKES Bundle ($TAG)" \
    --notes "Prebuilt apache-ctakes-6.0.0-bin with dictionary.\n\nSHA256: $SHA256"
fi

REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
ASSET_URL="https://github.com/${REPO}/releases/download/${TAG}/$(basename "$BUNDLE_FILE")"
echo "Published asset: $ASSET_URL"
echo "Tip: Set BUNDLE_URL='$ASSET_URL' when calling scripts/install_bundle.sh or rely on its default if using tag '$TAG'."

