# MIMIC Validation (100 Notes)

Goal: quickly validate your cTAKES instance and pipelines on a stable 100‑note sample before launching large runs.

What it does
- Builds a 100‑note subset from `samples/mimic/` (flat directory).
- Runs all compare pipelines with modest parallelism (default 4×4, 4 GB).
- Generates a lightweight manifest (hashes and counts) for each pipeline’s outputs.
- If a baseline manifest exists at `samples/mimic_output/manifest.txt`, compares against it.
- If no baseline exists, seeds one from the current run (no PHI included).

Usage
```
# 1) Place ~100 de‑identified MIMIC notes (.txt) under:
#    samples/mimic/

# 2) Run the validator (from repo root)
scripts/validate_mimic.sh

# Options:
#   -i|--in <dir>       source notes (default: samples/mimic)
#   -n|--count <N>      number of notes (default: 100)
#   -o|--out <dir>      outputs base (default: outputs/validation_mimic)
#   --runners N         runners per pipeline (default: 4)
#   --threads N         threads per runner (default: 4)
#   --xmx MB            heap per runner (default: 4096)
#   --seed VAL          sharding seed (default: 42)
#   --consolidate-async consolidate + report in background (default: off)

# 3) Interpret results
# - If baseline exists: VALIDATION OK / MISMATCH (diff shown)
# - If baseline missing: a new baseline manifest is created at samples/mimic_output/manifest.txt
```

What’s in the manifest?
- One line per pipeline run directory, for example:
```
[S_core_mimic_100_20250828-123456] docs=100 cui_count_hash=… bsv_table_hash=… tokens_hash=…
```
- Hashes are sha256 of sorted concatenation of the respective files; no raw note text is included.

Notes
- Do not commit raw notes under `samples/mimic/`. The repo’s `.gitignore` excludes `*.txt` in that folder.
- The baseline manifest is small and may be committed if you want a shared reference. Keep in mind different JVMs/OSes can cause minor variations; in that case, re‑seed per environment.
- The validator uses the same compare pipelines as large runs; it exercises dictionary, temporal models (if present), and writers.

