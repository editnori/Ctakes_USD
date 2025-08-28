# Full Bundle (Exact cTAKES Instance)

Goal: clone this repo, install one bundle, and run — no version drift. The bundle contains your exact cTAKES tree (including custom writers and the built dictionary).

Two steps
1) Build the bundle locally (one time):

```
# Put your exact cTAKES tree under: apache-ctakes-6.0.0-bin/
scripts/make_bundle.sh
```

This writes `CtakesBun-bundle.tgz` at repo root and prints a SHA256.

2) Publish a Release asset (recommended):

- Use the helper to publish a GitHub Release tagged `bundle` and upload the bundle:

```
scripts/publish_bundle_release.sh           # builds + uploads to tag 'bundle'
# or choose a different tag
scripts/publish_bundle_release.sh -t bundle-v1
```

- The script uses GitHub CLI (`gh`). Install from https://cli.github.com/ and run `gh auth login` once.

Consumers then run:

```
# Option A: local file at repo root (and install deps on Ubuntu/Debian)
scripts/install_bundle.sh --deps

# Option B: download from your release URL (default expects tag 'bundle')
scripts/install_bundle.sh --deps -u https://github.com/<owner>/<repo>/releases/download/bundle/CtakesBun-bundle.tgz -s <sha256>
```

After extraction, scripts default to `CTAKES_HOME=apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0` and you’re ready to run.

Notes
- Don’t commit the bundle to git (too large). Use Releases or external storage. `*.tgz` is already ignored.
- The bundle contains only `apache-ctakes-6.0.0-bin` (cTAKES and prebuilt dictionary). Pipelines, runners, and reporting stay in the repo so you can update them independently.
- Inputs (`SD*/`), logs, outputs, and `umls_loader/` are ignored by git and excluded from the bundle by design.
- If your bundle includes a local dictionary under `resources/org/apache/ctakes/dictionary/lookup/fast/`, the compare scripts will use it automatically.
- Regenerate a fresh bundle anytime after changing anything under `apache-ctakes-6.0.0-bin/`:

```
scripts/make_bundle.sh
scripts/publish_bundle_release.sh            # re-upload to the same tag
```

Dependencies (Ubuntu/Debian)
- `scripts/install_bundle.sh --deps` installs: `openjdk-17-jdk curl coreutils findutils gawk sed grep tar`.
- The script verifies `java`, `javac`, and `jar` are on PATH and prints their versions.
- Parallel runs stage the HSQL dictionary under `/dev/shm` for speed/locking; ensure it exists (it does on Ubuntu by default).
