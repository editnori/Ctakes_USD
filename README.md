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

   Writes outputs under `outputs/validate_mimic/` and compares a semantic manifest (concept CSVs, CUI counts, RxNorm rows, and XMI mentions) against the saved baseline when present. When run interactively, the script lets you choose which pipeline to validate (Core + Sectioned + Smoke combined pipeline by default).

5. **Run your own notes**

   ```bash

   bash scripts/run_pipeline.sh      --pipeline sectioned      -i /path/to/notes      -o /path/to/run_outputs

   ```

   Add `--with-relations` when you need relation extraction, or disable autoscale with `--no-autoscale` and supply explicit `--threads` / `--xmx` values.

   Or launch the interactive helper (`python scripts/ctakes_cli.py`) to pick the pipeline, discover note folders, and optionally detach via `--background` while recording progress metadata.
   When running in the background, check progress anytime with `python scripts/run_status.py --list` (or `--run <id>` for details).

6. **Inspect results**

   - `xmi/` contains CAS snapshots (one per note).

   - `concepts/` contains per-note CSVs written by `SimpleConceptCsvWriter`.

   - `cui_counts/` summarises CUI frequencies (separate affirmed/negated columns).

   - `rxnorm/` appears for the drug pipeline only.



> **Note**: The bundled FullClinical_AllTUIs dictionary ships inside the cTAKES archive. `scripts/run_pipeline.sh` automatically selects `resources/org/apache/ctakes/dictionary/lookup/fast/FullClinical_AllTUIs_local.xml` (or its non-local fallback) under `CTAKES_HOME`, so no manual dictionary configuration is required.

## Pipelines



| Key | Piper file | Purpose |
| --- | --- | --- |
| `core` | `pipelines/core/core_wsd.piper` | Default fast pipeline + dictionary lookup + WSD (add `--with-relations` for core relations). |
| `sectioned` | `pipelines/sectioned/sectioned_core_wsd.piper` | Section-aware pipeline with relations enabled by default. |
| `smoke` | `pipelines/smoke/sectioned_smoke_status.piper` | Sectioned pipeline with smoking-status annotators (relations optional via `--with-relations`). |
| `core_sectioned_smoke` | `pipelines/combined/core_sectioned_smoke.piper` | Combined sectioned pipeline with relations plus smoking annotators in one pass. |
| `drug` | `pipelines/drug/drug_ner_wsd.piper` | Sectioned pipeline with ctakes-drug-ner (adds RxNorm CSVs; add `--with-relations` for core relations). |



### Shared building blocks

| Area | Modules |
| --- | --- |
| Tokenization / POS / Chunking | `TsDefaultTokenizerPipeline` or `TsFullTokenizerPipeline`; `ContextDependentTokenizerAnnotator`; `POSTagger`; `TsChunkerSubPipe` |
| Dictionary lookup + WSD | `TsDictionarySubPipe`; `tools.wsd.SimpleWsdDisambiguatorAnnotator` (single-best concept without YTEX) |
| Assertion cleanup | `TsAttributeCleartkSubPipe`; `tools.fixes.DefaultSubjectAnnotator` (forces null subjects to `patient`) |
| Output writers | `FileTreeXmiWriter`; `tools.reporting.uima.SimpleConceptCsvWriter`; `tools.reporting.uima.CuiCountSummaryWriter`; `tools.reporting.uima.DrugRxNormCsvWriter` (drug pipeline only) |
| Smoking extras | `tools.smoking.SmokingAggregateStep1`; `tools.smoking.SmokingAggregateStep2Libsvm` |
| Drug extras | `tools.drug.DrugMentionAnnotatorWithTypes`; descriptor override `resources_override/.../DrugMentionAnnotator.xml` |

### Optional add-ons

| Flag | Piper insertion | Effect |
| --- | --- | --- |
| `--with-relations` | `load TsRelationSubPipe` | Adds core relation extraction to `core`, `smoke`, and `drug` before writers (ignored for pipelines that already include relations). |

## Script quick reference

| Script | Typical command | Notes |
| --- | --- | --- |
| `scripts/run_pipeline.sh` | `bash scripts/run_pipeline.sh --pipeline sectioned -i <notes> -o <outputs>` | Autosizes threads/heap, supports `--background` (nohup + log), recompiles helpers, accepts `--dict`, `--umls-key`, `--java-opts`, and `--with-relations` |
| `scripts/run_async.sh` | `bash scripts/run_async.sh --pipeline smoke -i <notes> -o <outputs>` | Shards input, runs multiple workers, shows progress/elapsed time, supports `--background`, and merges summary CSV/BSV outputs |
| `scripts/validate.sh` | `bash scripts/validate.sh --pipeline core_sectioned_smoke --limit 20 -i <notes> -o <run>` | Optional sampling (`--limit`), semantic manifest comparison (`--manifest`); add `--deterministic` to force single-thread mode |
| `scripts/validate_mimic.sh` | `bash scripts/validate_mimic.sh --pipeline drug` | Wrapper tuned for the bundled sample; creates per-pipeline subfolders and semantic manifest checks |
| `scripts/semantic_manifest.py` | `python scripts/semantic_manifest.py --outputs <run_dir> --manifest <manifest.json>` | Creates or compares the semantic manifest (concept CSVs, CUI counts, RxNorm rows, XMI mentions) |
| `scripts/flight_check.sh` | `bash scripts/flight_check.sh` | Verifies Java, CTAKES_HOME, sample data, and performs a dry run |
| `scripts/build_dictionary.sh` | `bash scripts/build_dictionary.sh --compile-only` | Compiles headless dictionary helpers; append `-- ...` to run `HeadlessDictionaryBuilder` |
| `scripts/ctakes_cli.py` | `python scripts/ctakes_cli.py` | Interactive runner to select pipeline/input, optionally mirror directories, and launch foreground/background runs (records metadata for run_status.py) |
| `scripts/run_status.py` | `python scripts/run_status.py --list` | Summarises recorded runs and reports progress using pipeline/async logs |

> **Runner logs:** `run_pipeline.sh` now emits `[pipeline][runner=N/M]` lines (threads, heap, start/finish) and honours `--background` to detach via `nohup`. `run_async.sh` mirrors the flag, shows per-shard progress/elapsed time, and writes shard logs under `<output>/.../shards/`.


## Validation workflows

| Scenario | Command | Outputs |
| --- | --- | --- |
| Full sample smoke test | `bash scripts/validate_mimic.sh` | Uses `samples/mimic`, writes to `outputs/validate_mimic/<pipeline>/`, and maintains per-pipeline semantic manifest JSONs (concept CSVs, CUI counts, RxNorm rows, XMI mentions) |
| Targeted validation | `bash scripts/validate.sh --pipeline smoke --limit 20 -i <notes> -o <run>` | Copies the first N notes to a temp workspace, runs the chosen pipeline, and optionally diffs against a semantic manifest (JSON) |
| Async batch run | `bash scripts/run_async.sh --pipeline sectioned -i <notes> -o <runs>` | Builds timestamped output folders and merged `concepts_summary.csv` / `rxnorm_summary.csv` (drug pipeline) |

> **Manifest tip:** Run `python scripts/semantic_manifest.py --outputs <run_dir> --manifest <manifest.json>` to create or compare the semantic manifest. The helper captures concept CSV rows, CUI counts, RxNorm rows, and XMI mentions in a stable order.
>
> ```bash
> python scripts/semantic_manifest.py --outputs outputs/validate_mimic/core --manifest samples/mimic_manifest_core.json
> ```
>
> Re-run the command after a new pipeline run to diff against the baseline; it prints the first mismatches and returns a non-zero status when semantics drift.

## Dictionary builder

| Mode | Command | Notes |
| --- | --- | --- |
| Compile only | `bash scripts/build_dictionary.sh --compile-only` | Writes classes to `build/dictionary/` (override with `BUILD_DIR`) |
| Compile + run | `bash scripts/build_dictionary.sh -- -u <umls> -o <dictionary>` | Passes arguments straight to `HeadlessDictionaryBuilder`; requires `CTAKES_HOME` and a UMLS key |

Set `CTAKES_HOME` before running, as the script adds `${CTAKES_HOME}/lib/*` to both the compile and runtime classpaths.

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
|-- resources_override/
|   |-- ctakes-drug-ner/desc/analysis_engine/DrugMentionAnnotator.xml
|   `-- org/apache/ctakes/drug/ner/types/TypeSystem.xml
```

`build/` and `outputs/` are transient. If a run misbehaves, remove them (`rm -rf build outputs`) and rerun; the scripts will rebuild the Java helpers automatically.

## Troubleshooting

| Symptom | Quick fix |
| --- | --- |
| RegexSpanFinder warnings | Bundled bundle includes slimmed dash regex; update to this release to silence the message. |
| `java` missing or < 11 | Install Java 11+ (Debian/Ubuntu: `bash scripts/install_deps.sh` or `sudo apt-get install openjdk-17-jdk`) |
| `CTAKES_HOME` warnings | Export `CTAKES_HOME` or let the bundled distro load; rerun `scripts/flight_check.sh` to persist the value into `.ctakes_env` |
| Dictionary XML not found | Ensure `${CTAKES_HOME}/resources/org/apache/ctakes/dictionary/lookup/fast/FullClinical_AllTUIs_local.xml` exists or pass `--dict <file>` |
| `validate.sh --limit` fails | Install `python3` / `python`, or run without `--limit` |
| Semantic groups look off | Update `resources/SemGroups.txt`; writers apply overrides via `SemGroupLoader` |

## Notes for future maintenance

| Area | Notes |
| --- | --- |
| Smoking pipeline | Local wrappers under `tools/smoking/` load the upstream `ctakes-smoking-status` aggregates |
| Drug pipeline | Depends on `tools.drug.DrugMentionAnnotatorWithTypes` and the override descriptor in `resources_override/.../DrugMentionAnnotator.xml` so the TypeSystem is present |
| Tool compilation | `scripts/run_pipeline.sh` recompiles Java helpers under `tools/` into `build/tools/` on every run |

That's it: a small toolkit that runs cTAKES pipelines, produces clean CSVs, and stays easy to follow.
