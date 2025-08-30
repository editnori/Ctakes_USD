# cTAKES Compare Runs

This repo runs Apache cTAKES at scale, writes the right per‑note artifacts during the run, then builds a modern Excel workbook in one step.

What you get
- One `.xlsx` workbook per run with: Overview, Pipeline Map, Processing Metrics, Clinical Concepts, CUI Counts, and Tokens.
- Per‑note “Clinical Concepts” CSVs (written during the run) for quick spot‑checks.
- XMI for each note (full record) if you need to drill down.

Clinician quick start (5 steps)
1) Install once (Ubuntu/Debian): see “First‑time install” below (2 commands).
2) Put ~100 sample notes under `samples/mimic/` and validate:
   - `scripts/validate_mimic.sh` (or `scripts/validate_mimic.sh --only S_core`)
   - You’ll see a small manifest to verify results. OK ⇒ proceed.
3) Set inputs and run at scale (fast defaults):
   - `export INPUT_ROOT=<path_to_notes>`
   - `export OUT_BASE=outputs/compare`
   - Either use autoscale (recommended):
     - `export SEED=42`
     - `bash scripts/run_compare_cluster.sh -i "$INPUT_ROOT" -o "$OUT_BASE" --reports --seed "$SEED" --autoscale`
   - Or set manual values:
     - `export RUNNERS=32 THREADS=8 XMX_MB=8192 SEED=42`
   - Optional shared dictionary cache (faster startup):
     - `bash scripts/prepare_shared_dict.sh -t /var/tmp/ctakes_dict_cache`
     - `export DICT_SHARED=1 DICT_SHARED_PATH=/var/tmp/ctakes_dict_cache`
   - Preview the plan:
     - `bash scripts/status.sh -i "$INPUT_ROOT" -o "$OUT_BASE"`
   - Run (autoscale):
     - `bash scripts/run_compare_cluster.sh -i "$INPUT_ROOT" -o "$OUT_BASE" --reports --seed "$SEED" --autoscale`
4) Monitor progress (any time):
   - `scripts/progress_compare_cluster.sh -i "$INPUT_ROOT" -o "$OUT_BASE"`
5) Open the workbook(s):
   - Per‑pipeline `.xlsx` files live in each pipeline run folder under `OUT_BASE`.

Prerequisites
- Java 11+
- cTAKES 6.0.0. Set `CTAKES_HOME` to `apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0` (or your install).
- Input notes under one directory (`.txt` files).

If you want a one‑shot install with your exact cTAKES build and dictionary, use the bundle workflow in `docs/BUNDLE.md`:

```
# Local file present (Ubuntu/Debian, also installs deps):
scripts/install_bundle.sh --deps

# Or download from your release URL:
scripts/install_bundle.sh --deps -u https://…/CtakesBun-bundle.tgz -s <sha256>
```

First‑time install (Ubuntu/Debian)
```
git clone https://github.com/editnori/Ctakes_USD.git CtakesBun
cd CtakesBun
scripts/install_bundle.sh --deps \
  -u https://github.com/editnori/Ctakes_USD/releases/download/bundle/CtakesBun-bundle.tgz \
  -s 0aae08a684ee5332aac0136e057cac0ee4fc29b34f2d5e3c3e763dc12f59e825
chmod +x scripts/*.sh
```

Updating to latest
```
cd <repo_root>
git fetch origin
git checkout main
git pull --ff-only origin main
```

Quick start
1) Start the run (detached optional):
   - `scripts/run_detached.sh scripts/run_compare_cluster.sh -i <input_dir> -o <output_base> --reports`
2) Consolidate and build the workbook:
   - `scripts/consolidate_shards.sh -p <run_dir> -W`
3) Open the workbook in Excel.

Inputs (example)
Your input folder can contain note‑type subfolders. Example with ~30,000 notes:

```
SD5000_1/
  AdmissionNote/
  DischargeSummary/
  EmergencyDepartmentNote/
  InpatientNote/
  OutpatientNote/
  RadiologyReport/
```

Progress and resume
- Progress: `scripts/progress_compare_cluster.sh -i SD5000_1 -o outputs/compare`
- Resume after stop: add `--resume` to the run command. It links only missing documents per shard (checks top‑level `xmi/`).
- Stable sharding: use `--seed 42` (any number) with the same `--runners` to keep the shard assignment stable across retries.
- Long runs: use `scripts/run_detached.sh …` to keep the job running if a terminal closes.

Review multiple runs
When you finish 10 runs and want one view:
- `scripts/build_multi_run_summary.sh -o <combined_dir> <run_dir1> ... <run_dir10>`
This links each run’s pipeline folders into `<combined_dir>` and builds `ctakes-runs-summary-<ts>.xlsx` in summary mode.

What happens behind the scenes
- Pipelines write per-note Clinical Concepts CSVs during the run (and CUI counts/tokens when configured).
- Consolidation (async when requested) moves shard outputs to top level, restores the tuned `.piper`, writes `run.log`, `timing_csv/timing.csv`, and a lightweight `metrics.json`, then removes shards.
- Parent compare workbook includes Pipelines Summary, Note Types Summary (aggregated by input group), and aggregate processing metrics.
- Report builds run in CSV mode by default (no XMI parsing).

Options you might change
- `--autoscale`: derive `RUNNERS/THREADS/XMX` from host cores and memory.
- `-n/--runners`, `-t/--threads`, `-m/--xmx`: manual parallelism and heap (watch memory).
- `--max-pipelines N`: run up to N pipelines concurrently (throttled at top-level).
- `--resume`: continue only missing documents.
- `--seed <val>`: keep shard assignment stable across runs (with the same `--runners`).

Repository layout
```
scripts/           # runners, consolidation, status, prepare-shared-dict, detached helper, multi-run summary
pipelines/         # compare pipelines and shared writer includes
tools/reporting/   # Excel workbook builder, CSV aggregator
tools/reporting/uima/ClinicalConceptCsvWriter.java  # in-pipeline per-note CSV writer
samples/mimic/     # place ~100 de-identified MIMIC notes (.txt) for validation
```

I ignore the cTAKES distro and runtime outputs in git. Keep those local.

## MIMIC Validation (cluster, lock-safe)

HSQL file DBs (the fast dictionary) cannot be opened concurrently at the same on‑disk path. To avoid `.lck` errors in parallel runs, enable dictionary relocation via `CTAKES_SANITIZE_DICT=1` (this copies the DB and rewrites the JDBC URL for each runner or a shared copy). This does not change dictionary content.

- `CTAKES_SANITIZE_DICT=1`: required to avoid HSQL `.lck` under parallelism (copies DB, rewrites URL).
- `DICT_SHARED=0`: per‑shard copies (each runner gets its own copy; more disk, less contention).
- `DICT_SHARED=1`: one shared read‑only copy for all runners (saves space; fastest startup). Set `DICT_SHARED_PATH` (e.g., `/dev/shm`).

Recommended per‑shard MIMIC validation run:

```bash
cd /workspace/CtakesBun

# 1. Set CTAKES_HOME
export CTAKES_HOME="$(pwd)/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"

# 2. Clean any existing locks
find "$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast" -name '*.lck' -delete

# 3. Env for per-shard copies + concurrency
export CTAKES_SANITIZE_DICT=1
export DICT_SHARED=0
# UMLS key is auto-injected by runners; override explicitly if needed
export UMLS_KEY="6370dcdd-d438-47ab-8749-5a8fb9d013f2"
ulimit -n 65535 || true

# 4. Run validation (reuse when you already have .txt files present)
scripts/validate_mimic.sh \
  -i "/workspace/CtakesBun/samples/mimic" \
  -o "/workspace/CtakesBun/outputs/mimic_validation" \
  --runners 32 \
  --threads 8 \
  --xmx 8192 \
  --seed 42 \
  --subset-mode reuse \
  --consolidate-async
```

Tip: Prefer a single shared copy to save space:

```bash
export CTAKES_SANITIZE_DICT=1
export DICT_SHARED=1
export DICT_SHARED_PATH=/dev/shm
bash scripts/flight_check.sh --mode cluster --require-shared
scripts/validate_mimic.sh -i "/workspace/CtakesBun/samples/mimic" -o "/workspace/CtakesBun/outputs/mimic_validation" --subset-mode reuse --consolidate-async
```

Reduce XMI warning verbosity (optional):

```bash
# Prior to run_compare_cluster.sh invocations (propagates from validate_mimic)
export XMI_LOG_LEVEL=error
```

About the XMI “multipleReferencesAllowed” warnings:
- These are benign serializer warnings when a feature is referenced multiple times but the TypeSystem marks it as not allowing multiple references (e.g., `Predicate:relations`).
- Suppress by setting `XMI_LOG_LEVEL=error`.
- Advanced fix (optional) is to override the TypeSystem feature to set `<multipleReferencesAllowed>true</multipleReferencesAllowed>`, which requires a runtime TypeSystem override.

All commands (reference)
- `scripts/run_compare_cluster.sh`
  - `-i|--in <dir>` input root (supports note‑type subfolders like the example above).
  - `-o|--out <dir>` output base.
  - `-n|--runners <N>` parallel runners per pipeline (default 16).
  - `-t|--threads <N>` threads per runner (default 6).
  - `-m|--xmx <MB>` heap per runner (default 6144).
  - `--seed <val>` stable sharding seed.
  - `--resume` resume only missing docs (checks top‑level `xmi/`).
  - `--reports` build per‑pipeline reports during run (CSV mode; async with `--reports-async`).
  - `--no-consolidate` keep `shard_*` (normally removed during consolidation).
  - `--consolidate-async` queue consolidation/report jobs and wait for them at the end.
  - `--keep-shards` consolidate but retain `shard_*` and `shards/` directories.
  - `--only "<keys>"` run only specific pipelines (e.g., `"S_core D_core_temp"`).
  - `-l|--dict-xml <file>` override dictionary XML descriptor.
  - `--autoscale` derive `RUNNERS/THREADS/XMX` from host cores and memory (fast default).
  - `--max-pipelines <N>` run up to N pipeline-group tasks concurrently at the top level.
  - `--reports-sync` build per‑pipeline report synchronously; `--reports-async` to run in background.
  - `--no-parent-report` skip building the parent compare workbook at the OUT base.

- `scripts/status.sh` — dry-run status of what would be executed.
  - `-i|--in <dir>` input root.
  - `-o|--out <dir>` output base (default `outputs/compare`).
  - `--only` limit pipelines; shows which will run.
  - Prints environment (RUNNERS/THREADS/XMX), dictionary + shared cache status, and report mode.

- `scripts/consolidate_shards.sh`
  - `-p|--parent <run_dir>` required.
  - `--keep-shards` keep shards; default removes `shard_*`, `shards/`, any `pending_*`.
  - `-W|--workbook [path]` build workbook (defaults to `<run>/ctakes-report-<ts>.xlsx`).
  - `--wb-mode <summary|csv|full>` report mode. `csv` is the fast default for `-W`.

- `scripts/build_xlsx_report.sh`  — build a workbook from outputs.
  - `-o|--out <run_dir>` required. `-w|--workbook <file>` optional. `-M|--mode <summary|csv|full>`.

- `scripts/progress_compare_cluster.sh` — progress estimator.
  - `-i|--in <input_root>` and `-o|--out <output_base>`.

- `scripts/run_detached.sh` — run any script under `nohup`, write log + PID to `logs/`.

- `scripts/build_multi_run_summary.sh` - one summary across multiple runs.

Deprecated scripts (kept for reference)
- `scripts/deprecated/ctakes_cluster_run.sh` – use `scripts/run_compare_cluster.sh`.
- `scripts/deprecated/build_report.sh` – use `scripts/build_xlsx_report.sh`.
- `scripts/deprecated/consolidate_cuicount.py` – use the CuiCounts sheet in the Java workbook.

## Copy-Paste Commands (Ubuntu/Debian)

These are ready to run from the repository root.

### 1. Update Repo to Latest Main

Non-destructive (fast-forward only):

```bash
cd <repo_root>
git fetch origin
git checkout main
git pull --ff-only origin main
```

Discard local changes (force update):

```bash
cd <repo_root>
git fetch origin main
git reset --hard origin/main
```

### 2. Fix Script Permissions (one-time)

```bash
chmod +x scripts/*.sh
```

### 3. Quick Validation (100-note sample)

```bash
# Place ~100 .txt files in samples/mimic/ first, then:
scripts/validate_mimic.sh
# Results will be compared/seeded at samples/mimic_output/manifest.txt
# To accept updated expected outputs, re-run with:
#   scripts/validate_mimic.sh --update-baseline
```

### 4. Run Large Compare (CSV-mode reports, autoscale)

```bash
export INPUT_ROOT="<path_to_notes>"
export OUT_BASE="outputs/compare"
export RUNNERS=32
export THREADS=8
export XMX_MB=8192
export SEED=42
ulimit -n 65535 || true

# Optional: shared dictionary cache for faster startup
bash scripts/prepare_shared_dict.sh -t /var/tmp/ctakes_dict_cache
export DICT_SHARED=1
export DICT_SHARED_PATH=/var/tmp/ctakes_dict_cache

bash scripts/status.sh -i "$INPUT_ROOT" -o "$OUT_BASE"
bash scripts/run_compare_cluster.sh -i "$INPUT_ROOT" -o "$OUT_BASE" --reports --seed "$SEED" --autoscale
```

### 5. Check Progress and Outputs

Check overall progress:

```bash
scripts/progress_compare_cluster.sh -i "$INPUT_ROOT" -o "$OUT_BASE"
```

Monitor live logs (replace `*` with actual run directory once created):

```bash
tail -n 50 -F "$OUT_BASE"/*/shard_*/run.log
```

List generated Excel workbooks:

```bash
ls -1 "$OUT_BASE"/*/ctakes-*.xlsx
```

Sanity-check input distribution (per subfolder):

```bash
for dir in "$INPUT_ROOT"/*/; do 
  [ -d "$dir" ] || continue
  count=$(find "$dir" -type f -name "*.txt" | wc -l)
  [ "$count" -gt 0 ] && echo "$(basename "$dir"): $count files"
done
```

Watch overall XMI progress and latest file (10s interval):

```bash
watch -n 10 '
  total=$(find "$INPUT_ROOT" -type f -name "*.txt" | wc -l);
  done=$(find "$OUT_BASE" -type f -name "*.xmi" | wc -l);
  pct=$((done*100/ (total>0?total:1) ));
  echo "Progress: $done/$total (${pct}%)";
  echo "Files remaining: $((total-done))";
  ls -lt "$OUT_BASE"/*/*.xmi 2>/dev/null | head -1
'
```

Example progress output:

```
Input notes:        30000
Pipelines planned:   10 (S_core S_core_rel D_core_rel D_core_coref S_core_temp S_core_temp_coref D_core_temp D_core_temp_coref S_core_temp_coref_smoke D_core_temp_coref_smoke)
Expected XMI total:  300000
Current XMI count:   504
Progress (XMI):      0.17%

Expected all files:  2100000  (7 per doc per pipeline)
Current all files:   3054
Progress (all types): 0.15%
```

### 6. Install Bundle on New Machine

```bash
scripts/install_bundle.sh --deps \
  -u https://github.com/editnori/Ctakes_USD/releases/download/bundle/CtakesBun-bundle.tgz \
  -s 0aae08a684ee5332aac0136e057cac0ee4fc29b34f2d5e3c3e763dc12f59e825
```

### All-in-One Main Workflow

```bash
# 1) Update repo
cd <repo_root>
git fetch origin
git checkout main
git pull --ff-only origin main

# 2) Fix permissions if needed
chmod +x scripts/*.sh

# 3) Set environment variables
export INPUT_ROOT="<path_to_notes>"
export OUT_BASE="outputs/compare"
export RUNNERS=32
export THREADS=8
export XMX_MB=8192
export SEED=42
ulimit -n 65535 || true

# Optional: shared dictionary cache
bash scripts/prepare_shared_dict.sh -t /var/tmp/ctakes_dict_cache
export DICT_SHARED=1
export DICT_SHARED_PATH=/var/tmp/ctakes_dict_cache

# 4) Run the comparison (async consolidate + async reports; autoscale)
bash scripts/status.sh -i "$INPUT_ROOT" -o "$OUT_BASE"
bash scripts/run_compare_cluster.sh -i "$INPUT_ROOT" -o "$OUT_BASE" --reports --seed "$SEED" --autoscale

# 5) Check progress in another terminal
scripts/progress_compare_cluster.sh -i "$INPUT_ROOT" -o "$OUT_BASE"
```

Validation (100‑note MIMIC sample)
- Place ~100 `.txt` notes under `samples/mimic/`
- Run: `scripts/validate_mimic.sh` (or `--only S_core`)
- Subset handling: `--subset-mode reuse|link|copy` (default: link). Auto‑reuse when input folder already has exactly `COUNT` notes.
- Compares against `samples/mimic_output/manifest.txt` if present, or seeds it on first run.

## Pipelines: What Runs by Default

By default the runner executes a set of pipelines; if Temporal models are found, you get 10 pipelines, otherwise 4 (non‑temporal).

Key prefixes
- `S_`: Section‑aware (TsFullTokenizerPipeline). Keeps section boundaries for downstream AEs.
- `D_`: Default core (TsDefaultTokenizerPipeline). No explicit section handling.

Suffixes
- `_core`: Core NLP + Dictionary + WSD + Assertion + Writers.
- `_rel`: Adds clinical relations (degree/location/modifier) via TsRelationSubPipe.
- `_temp`: Adds temporal events/links via THYME classifiers (requires models).
- `_coref`: Adds coreference resolution (Markable chains).
- `_smoke`: Adds Smoking Status classification AEs (rule‑based + PCS).

Pipelines (keys → file → summary)
- `S_core` → `pipelines/compare/TsSectionedFast_WSD_Compare.piper` — Section‑aware tokenization, dictionary lookup, WSD, assertion, unified writers.
- `S_core_rel` → `pipelines/compare/TsSectionedRelation_WSD_Compare.piper` — `S_core` + clinical relations (degree/location modifiers).
- `S_core_temp` → `pipelines/compare/TsSectionedTemporal_WSD_Compare.piper` — `S_core` + temporal events/relations (THYME models).
- `S_core_temp_coref` → `pipelines/compare/TsSectionedTemporalCoref_WSD_Compare.piper` — `S_core_temp` + coreference resolution.
- `S_core_temp_coref_smoke` → `pipelines/compare/TsSectionedTemporalCoref_WSD_Smoking_Compare.piper` — `S_core_temp_coref` + Smoking Status annotators.
- `D_core_rel` → `pipelines/compare/TsDefaultRelation_WSD_Compare.piper` — Default tokenizer `D_core` + relations.
- `D_core_temp` → `pipelines/compare/TsDefaultTemporal_WSD_Compare.piper` — `D_core` + temporal events/relations.
- `D_core_temp_coref` → `pipelines/compare/TsDefaultTemporalCoref_WSD_Compare.piper` — `D_core_temp` + coreference.
- `D_core_temp_coref_smoke` → `pipelines/compare/TsDefaultTemporalCoref_WSD_Smoking_Compare.piper` — `D_core_temp_coref` + Smoking Status.
- `D_core_coref` → `pipelines/compare/TsDefaultCoref_WSD_Compare.piper` — `D_core` + coreference (no temporal, no relations).

What “Core” does
- Tokenize, POS tag, chunk; dictionary lookup (fast HSQL rare‑word index); WSD (`tools.wsd.SimpleWsdDisambiguatorAnnotator` picks one best); assertion features (polarity/uncertainty/conditional/generic/subject) with a safety default subject (patient).
- Writers produce: XMI, `bsv_table/`, `csv_table/`, `html_table/`, `cui_list/`, `cui_count/`, `bsv_tokens/`, and `csv_table_concepts/` (per‑doc Clinical Concepts with full columns).

Select a single pipeline (or a subset)
- One pipeline: `scripts/run_compare_cluster.sh -i "$INPUT_ROOT" -o "$OUT_BASE" --only S_core --reports`
- Multiple: `scripts/run_compare_cluster.sh -i "$INPUT_ROOT" -o "$OUT_BASE" --only "S_core D_core_temp" --reports`
- Temporal only (if models present): `scripts/run_compare_cluster.sh -i "$INPUT_ROOT" -o "$OUT_BASE" --only "S_core_temp D_core_temp" --reports`
