# cTAKES USD Clean Toolkit

This repository packages a small, predictable toolkit on top of Apache cTAKES 6.x. It keeps only the pieces we need in day-to-day runs: a core set of pipelines, lean CSV writers, and a few helper scripts. Everything else (old compare workflows, report generators, archived outputs) has been removed so the layout stays easy to follow.

## What you get

- **Four ready-to-run pipelines** (`core`, `sectioned`, `smoke`, `drug`) with a shared OPTIONAL_MODULES hook so temporal/coref components can be toggled on demand.
- **Lean output writers** that emit CAS XMI, per-document concept CSVs, CUI counts, and (for the drug pipeline) RxNorm rows.
- **Autoscale-friendly runners** (`run_pipeline.sh`, `run_async.sh`) that compile the local Java helpers, size heap/threads sensibly, and wire in optional modules.
- **Validation helpers** (`validate.sh`, `validate_mimic.sh`) plus a quick flight check to make sure prerequisites are in place.
- **Headless dictionary tooling** for building or refreshing the UMLS dictionary bundle.
- **Bundled cTAKES distribution** under `CtakesBun-bundle/` so you can get started without installing cTAKES separately.

## Quick start

1. **Clone the toolkit**
   ```bash
   git clone https://github.com/editnori/Ctakes_USD.git
   cd Ctakes_USD
   ```
2. **Bootstrap dependencies and bundle**
   - Debian/Ubuntu: `bash scripts/setup.sh --deps` (installs packages and downloads the cTAKES bundle).
   - Other environments: run `bash scripts/install_deps.sh` (or install Java 11+, curl, tar, unzip, python3 manually) and `bash scripts/get_bundle.sh`.
3. **Run a health check**
   ```bash
   bash scripts/flight_check.sh
   ```
   Confirms Java, the bundled cTAKES install, pipeline files, and performs a dry-run on the sample notes when available. When run interactively, the script offers to persist CTAKES_HOME and the default UMLS API key into `.ctakes_env` so subsequent scripts pick up those settings automatically.
4. **Smoke test on the bundled 100 notes**
   ```bash
   bash scripts/validate_mimic.sh
   ```
   Writes outputs under `outputs/validate_mimic/` and compares hashes against `samples/mimic_manifest.txt` when present.
5. **Run your own notes**
   ```bash
   bash scripts/run_pipeline.sh      --pipeline sectioned      --autoscale      -i /path/to/notes      -o /path/to/run_outputs
   ```
   Add `--with-temporal` and/or `--with-coref` as needed, or disable autoscale with `--no-autoscale` and supply explicit `--threads` / `--xmx` values.
6. **Inspect results**
   - `xmi/` contains CAS snapshots (one per note).
   - `concepts/` contains per-note CSVs written by `SimpleConceptCsvWriter`.
   - `cui_count/` summarises CUI frequencies.
   - `rxnorm/` appears for the drug pipeline only.

> **Note**: The bundled FullClinical_AllTUIs dictionary ships inside the cTAKES archive. `scripts/run_pipeline.sh` automatically selects `resources/org/apache/ctakes/dictionary/lookup/fast/FullClinical_AllTUIs_local.xml` (or its non-local fallback) under `CTAKES_HOME`, so no manual dictionary configuration is required.
## Pipelines

| Key | Piper file | Purpose |
| --- | --- | --- |
| `core` | `pipelines/core/core_wsd.piper` | Default fast pipeline + dictionary lookup + WSD. |
| `sectioned` | `pipelines/sectioned/sectioned_core_wsd.piper` | Section-aware pipeline with relations. |
| `smoke` | `pipelines/smoke/sectioned_smoke_status.piper` | Sectioned pipeline with smoking-status annotators. |
| `drug` | `pipelines/drug/drug_ner_wsd.piper` | Sectioned pipeline with ctakes-drug-ner (adds RxNorm CSVs). |

### What runs inside each pipeline

All four pipelines load the same core building blocks from the cTAKES distribution and add a couple of local helpers:

- **Tokenization / POS / Chunking**: `TsDefaultTokenizerPipeline` or `TsFullTokenizerPipeline`, `ContextDependentTokenizerAnnotator`, `POSTagger`, `TsChunkerSubPipe`.
- **Dictionary lookup + WSD**: `TsDictionarySubPipe` followed by our `tools.wsd.SimpleWsdDisambiguatorAnnotator` (single-best concept without YTEX).
- **Assertion cleanup**: `TsAttributeCleartkSubPipe` and `tools.fixes.DefaultSubjectAnnotator` (forces `IdentifiedAnnotation#subject` to "patient" when null).
- **Writers**: `FileTreeXmiWriter`, `tools.reporting.uima.SimpleConceptCsvWriter`, `CuiCountFileWriter`, and for the drug pipeline `tools.reporting.uima.DrugRxNormCsvWriter` (honours overrides from `resources/SemGroups.txt`).
- **Smoking pipeline extras**: the aggregate wrappers `tools.smoking.SmokingAggregateStep1` and `SmokingAggregateStep2Libsvm`, which delegate to the bundled `ctakes-smoking-status` descriptors.
- **Drug pipeline extras**: the override descriptor `resources_override/.../DrugMentionAnnotator_WithTypes.xml` to ensure the drug TypeSystem is available.

### Run a pipeline

```bash
bash scripts/run_pipeline.sh   --pipeline sectioned   --with-temporal   --with-coref   --autoscale   -i /data/notes   -o /runs/sectioned_temporal
```

Highlights:
- `--autoscale` inspects CPU and RAM and applies sensible defaults (threads roughly cores/2, heap about 60% of RAM, clamped between 2 and 24 GB).
- `--threads <N>` or `--xmx <MB>` override the autoscale choices.
- `--dry-run` prints the Java command without executing it.
- If `CTAKES_HOME` is unset and the bundled cTAKES exists, the script uses it automatically.

### Run asynchronously

```bash
bash scripts/run_async.sh   --pipeline smoke   -i /data/notes   -o /runs/smoke_async
```

`run_async.sh` partitions the input notes across multiple shards, launches `run_pipeline.sh` for each shard in parallel, then consolidates outputs into `<output>/<pipeline>/<timestamp>/`:

- `xmi/`, `concepts/`, `cui_count/`, and optionally `rxnorm/`.
- `concepts_summary.csv` (and `rxnorm_summary.csv` when applicable) built by concatenating the shard CSVs.

Use `--dry-run` to inspect the planned per-shard commands. Manual overrides (`--shards`, `--threads`, `--xmx`, `--dict`, etc.) pass straight through to the child runs.

## Validation workflows

### Ad-hoc sampling

```bash
bash scripts/validate.sh   --pipeline smoke   --limit 20   -i /data/notes   -o /runs/validate_smoke
```

`--limit` copies the first N `.txt/.xmi/.xml` files into a temporary folder (requires `python3` or `python`). Omit `--limit` to process the full directory.

### Bundled 100-note smoke test

```bash
bash scripts/validate_mimic.sh
```

`--with-temporal`, `--with-coref`, and `--dry-run` pass straight through to `scripts/validate.sh`.

### Manifest notes

Both validation scripts can compare their outputs against a saved manifest via `--manifest <file>`. The manifest is a simple `sha256sum`-style file (`<hash>  <relative-path>` on each line). On Windows you can generate one with either GNU coreutils or PowerShell:

```powershell
Get-ChildItem outputs/validate_smoke -Recurse -File |
  ForEach-Object {
    $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLower()
    $rel  = Resolve-Path $_ -Relative
    "$hash  $rel"
  } | Set-Content -Encoding utf8 samples/mimic_manifest.txt
```

## Dictionary builder

```bash
# Compile classes only (drops .class files in build/dictionary/)
bash scripts/build_dictionary.sh --compile-only

# Compile + run (everything after "--" is passed straight to HeadlessDictionaryBuilder)
bash scripts/build_dictionary.sh -- -u /path/to/UMLS -o /path/to/dictionary
```

Set `BUILD_DIR=/custom/path` to override the output location. The script automatically adds `${CTAKES_HOME}/lib/*` to the classpath.

## Repository layout (high level)

```
.
|-- CtakesBun-bundle/                      # Clean apache-cTAKES distribution
|-- pipelines/
|   |-- core/core_wsd.piper
|   |-- sectioned/sectioned_core_wsd.piper
|   |-- smoke/sectioned_smoke_status.piper
|   `-- drug/drug_ner_wsd.piper
|-- scripts/
|   |-- build_dictionary.sh
|   |-- flight_check.sh
|   |-- run_pipeline.sh
|   |-- run_async.sh
|   |-- validate.sh
|   `-- validate_mimic.sh
|-- tools/
|   |-- HeadlessDictionary*.java
|   |-- fixes/DefaultSubjectAnnotator.java
|   |-- reporting/uima/*.java
|   |-- smoking/SmokingAggregateStep{1,2}Libsvm.java
|   `-- wsd/SimpleWsdDisambiguatorAnnotator.java
`-- resources_override/org/apache/ctakes/drugner/ae/DrugMentionAnnotator_WithTypes.xml
```

`build/` and `outputs/` are transient. If a run misbehaves, remove them (`rm -rf build outputs`) and rerun; the scripts will rebuild the Java helpers automatically.

## Troubleshooting

- `scripts/flight_check.sh` points its dry run at `samples/mimic/` and skips it when no notes are available. A warning usually means the bundled dictionary or sample set is missing.
- Delete `build/` and `outputs/` if you need to clear stale CAS files or compiled classes.
- Check `resources/SemGroups.txt` when semantic group labels look off; the writers merge it into the cTAKES defaults.
- `java.lang.NoClassDefFoundError`: ensure Java 11+ is in use and `${CTAKES_HOME}/lib` contains the cTAKES jars.
- `python not found` when using `validate.sh --limit`: install Python (`python3` or `python`) or run without `--limit`.

## Notes for future maintenance

- The smoking pipeline uses the bundled cTAKES aggregates. The only local code under `tools/smoking/` are the wrappers that load the production aggregate descriptors.
- The drug pipeline depends on `resources_override/.../DrugMentionAnnotator_WithTypes.xml` so the drug TypeSystem is on the classpath.
- `scripts/run_pipeline.sh` recompiles everything under `tools/` each time it runs and writes classes to `build/tools/`.

ThatÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢s it: a small toolkit that runs cTAKES pipelines, produces clean CSVs, and stays easy to follow.
