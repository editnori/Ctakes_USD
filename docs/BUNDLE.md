# Full Bundle (Exact cTAKES Instance)

Goal: clone this repo, install one bundle, and run — no version drift. The bundle contains your exact cTAKES tree (including custom writers and the built dictionary).

Two steps
1) Build the bundle locally (one time):

```
# Put your exact cTAKES tree under: apache-ctakes-6.0.0-bin/
scripts/make_bundle.sh
```

This writes `CtakesBun-bundle.tgz` at repo root and prints a SHA256.

2) Host it somewhere and install:

- Recommended: upload `CtakesBun-bundle.tgz` as a GitHub Release asset (e.g., tag `bundle`).
- Or host on S3/Drive. Copy the URL.

Consumers then run:

```
# Option A: local file at repo root
scripts/install_bundle.sh

# Option B: download from a URL
scripts/install_bundle.sh -u https://…/CtakesBun-bundle.tgz -s <sha256>
```

After extraction, scripts default to `CTAKES_HOME=apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0` and you’re ready to run.

Notes
- Don’t commit the bundle to git (too large). Use Releases or external storage.
- If your bundle includes a local dictionary under `resources/org/apache/ctakes/dictionary/lookup/fast/`, the compare scripts will use it automatically.
- You can regenerate a fresh bundle any time with `scripts/make_bundle.sh`.

