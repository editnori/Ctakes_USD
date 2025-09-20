# cTAKES USD Clean Toolkit

This repository packages a small, predictable toolkit on top of Apache cTAKES 6.x. It keeps only the pieces we need in day-to-day runs: a core set of pipelines, lean CSV writers, and a few helper scripts. Everything else (old compare workflows, report generators, archived outputs) has been removed so the layout stays easy to follow.

## What you get

- **Four ready-to-run pipelines** (`core`, `sectioned`, `smoke`, `drug`) with a shared OPTIONAL_MODULES hook so temporal/coref components can be toggled on demand.

- **Lean output writers** that emit CAS XMI, per-document concept CSVs, CUI counts, and (for the drug pipeline) RxNorm rows.

- **Autoscale-friendly runners** (`run_pipeline.sh`, `run_async.sh`) that compile the local Java helpers, size heap/threads sensibly, and wire in optional modules.

- **Validation helpers** (`validate.sh`, `validate_mimic.sh`) plus a quick flight check to make sure prerequisites are in place.

- **Clinical concept upgrades** covering automated dictionary rebuilds (discovery-driven SAB/TUI merges, RxNorm augmentation, timestamped logs), zero-dependency WSD, and thread-safe relation extraction tuned for multi-threaded runs.

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

4. **Smoke test with a local 100-note pack (optional)**

   ```bash

   bash scripts/validate_mimic.sh

   ```

   Drop a de-identified sample pack under `samples/mimic/` (the repo ignores the `.txt` files; no notes ship with the repo or release assets). The helper skips the run when the directory is empty so you can keep the toolkit clean between releases. When notes are present it writes outputs under `outputs/validate_mimic/` and compares a semantic manifest (concept CSVs, CUI counts, RxNorm rows, and XMI mentions) against the saved baseline. When run interactively, the script lets you choose which pipeline to validate (Core + Sectioned + Smoke combined pipeline by default).

5. **Run your own notes**

   ```bash

   bash scripts/run_pipeline.sh      --pipeline sectioned      -i /path/to/notes      -o /path/to/run_outputs

   ```

   Add `--with-relations` when you need relation extraction, or disable autoscale with `--no-autoscale` and supply explicit `--threads` / `--xmx` values.

   Or launch the interactive helper (`python scripts/ctakes_cli.py`) to pick the pipeline, discover note folders, and optionally detach via `--background` while recording progress metadata.
   When running in the background, check progress anytime with `python scripts/run_status.py --list` (or `--run <id>` for details).

6. **Inspect results**

   - `xmi/` contains CAS snapshots (one per note).

   - `concepts/` contains per-note CSVs written by `ConceptCsvWriter`.
   - `html/` hosts interactive note views when `HtmlAnnotationOverviewWriter` is enabled. Use it to toggle layers (core, WSD, drug, relations, coref) and inspect how annotations overlap.

   - `cui_counts/` summarises CUI frequencies (separate affirmed/negated columns).

  - `rxnorm/` appears for the drug pipeline only (Document,Begin,End,Text,Section,CUI,RxCUI,RxNormName,TUI,SemanticGroup,SemanticTypeLabel).

> **Note**: The bundled KidneyStone_SDOH dictionary ships inside the cTAKES archive. `scripts/run_pipeline.sh` automatically selects `resources/org/apache/ctakes/dictionary/lookup/fast/KidneyStone_SDOH_local.xml` (or its non-local fallback) under `CTAKES_HOME`, so no manual dictionary configuration is required.

## Pipelines

| Key | Piper file | Purpose |
| --- | --- | --- |
| `core` | `pipelines/core/core_wsd.piper` | Default fast pipeline + dictionary lookup + WSD (add `--with-relations` for core relations). |
| `s_core_relations_smoke` | `pipelines/combined/s_core_relations_smoke.piper` | Sectioned combined pipeline with filtered core relations (fast default in helper scripts). |
| `sectioned` | `pipelines/sectioned/sectioned_core_wsd.piper` | Section-aware pipeline with relations enabled by default. |
| `smoke` | `pipelines/smoke/sectioned_smoke_status.piper` | Sectioned pipeline with smoking-status annotators (relations optional via `--with-relations`). |
| `core_sectioned_smoke` | `pipelines/combined/core_sectioned_smoke.piper` | Combined sectioned pipeline with relations plus smoking annotators in one pass. |
| `drug` | `pipelines/drug/drug_ner_wsd.piper` | Sectioned pipeline with ctakes-drug-ner (adds RxNorm CSVs with CUI+RxCUI; add `--with-relations` for core relations). |

Each Piper descriptor pins `threads 3`; `scripts/run_pipeline.sh` rewrites the value when you pass `--threads` or enable `--autoscale`, keeping the runtime thread count aligned with the CLI flags.

The helper scripts default to `s_core_relations_smoke`; choose `core_sectioned_smoke` when you need full relation coverage without filtering.

### Shared building blocks

| Area | Modules |
| --- | --- |
| Tokenization / POS / Chunking | `TsDefaultTokenizerPipeline` or `TsFullTokenizerPipeline`; `ContextDependentTokenizerAnnotator`; `POSTagger`; `TsChunkerSubPipe` |
| Dictionary lookup + WSD | `TsDictionarySubPipe`; `tools.wsd.SimpleWsdDisambiguatorAnnotator` (single-best concept without YTEX) |
| Assertion cleanup | `TsAttributeCleartkSubPipe`; `tools.assertion.DefaultSubjectAnnotator` (forces null subjects to `patient`) |
| Output writers | `FileTreeXmiWriter`; `tools.reporting.uima.ConceptCsvWriter`; `tools.reporting.uima.CuiCountSummaryWriter`; `tools.reporting.uima.DrugRxNormCsvWriter` (drug pipeline only); `tools.reporting.uima.HtmlAnnotationOverviewWriter` |
| Smoking extras | `tools.smoking.SmokingAggregateStep1`; `tools.smoking.SmokingAggregateStep2Libsvm` |
| Drug extras | `tools.drug.DrugMentionAnnotatorWithTypes`; the bundled descriptor already includes the TypeSystem so Piper can resolve it |

### Optional add-ons

| Flag | Piper insertion | Effect |
| --- | --- | --- |
| `--with-relations` | `load TsRelationSubPipe` | Adds the fast relation sub-pipeline (modifier/degree models plus `tools.relations.ThreadSafeFastLocationExtractor`) to `core`, `smoke`, and `drug`; ignored when relations already present. |

### Simple WSD disambiguator

- `tools.wsd.SimpleWsdDisambiguatorAnnotator` picks a single best UMLS concept per mention using sentence-level token overlap (see `tools/wsd/SimpleWsdDisambiguatorAnnotator.java`).
- Default parameters keep the original candidate array, move the winner to the front, and stamp confidence/flags; adjust via Piper keys `KeepAllCandidates`, `MoveBestFirst`, `MarkDisambiguated`, `MinTokenLen`, and `FilterSingleCharStops`.
- Runs immediately after `TsDictionarySubPipe` in every pipeline; remove the `add tools.wsd.SimpleWsdDisambiguatorAnnotator ...` line if you prefer raw dictionary candidates.
- Emits normalized context scores so downstream writers can audit disambiguation quality without external services.

### Relation extraction upgrades

- Release bundles include `org/apache/ctakes/relation/extractor/pipeline/TsFastRelationSubPipe.piper`, which swaps in `tools.relations.ThreadSafeFastLocationExtractor` alongside the modifier and degree extractors. Re-run `scripts/get_bundle.sh` if your local bundle predates the change and is missing the descriptor.
- `tools.relations.FastLocationOfRelationExtractor` filters location-of candidates more than 30 tokens apart; tune with `-Dctakes.relations.max_token_distance=<n>` when launching pipelines.
- The thread-safe wrapper reuses a shared delegate so multi-thread runs avoid repeated ClearTK model loads.
- `--with-relations` in `scripts/run_pipeline.sh` injects this sub-pipeline for `core`, `smoke`, and `drug`; combined pipelines already include it.

## Script quick reference

| Script | Typical command | Notes |
| --- | --- | --- |
| `scripts/run_pipeline.sh` | `bash scripts/run_pipeline.sh --pipeline sectioned -i <notes> -o <outputs>` | Autosizes threads/heap, supports `--background` (nohup + log), recompiles helpers, accepts `--dict`, `--umls-key`, `--java-opts`, and `--with-relations` |
| `scripts/run_async.sh` | `bash scripts/run_async.sh --pipeline smoke -i <notes> -o <outputs>` | Shards input, runs multiple workers, shows progress/elapsed time, supports `--background`, and merges summary CSV/BSV outputs |
| `scripts/validate.sh` | `bash scripts/validate.sh --pipeline core_sectioned_smoke --limit 20 -i <notes> -o <run>` | Optional sampling (`--limit`), semantic manifest comparison (`--manifest`); add `--deterministic` to force single-thread mode |
| `scripts/validate_mimic.sh` | `bash scripts/validate_mimic.sh --pipeline drug` | Wrapper for an optional sample pack (no notes bundled); creates per-pipeline subfolders when notes are present and replays semantic manifest checks |
| `scripts/semantic_manifest.py` | `python scripts/semantic_manifest.py --outputs <run_dir> --manifest <manifest.json>` | Creates or compares the semantic manifest (concept CSVs, CUI counts, RxNorm rows, XMI mentions) |
| `scripts/flight_check.sh` | `bash scripts/flight_check.sh` | Verifies Java, CTAKES_HOME, sample data, and performs a dry run |
| `scripts/build_dictionary.sh` | `bash scripts/build_dictionary.sh --compile-only` | Compiles headless dictionary helpers; append `-- ...` to run `tools.dictionary.HeadlessDictionaryBuilder` |
| `scripts/build_dictionary_full.sh` | `bash scripts/build_dictionary_full.sh` | Discovers SAB/TUI values, normalises the UMLS snapshot, writes merged properties/logs, and rebuilds KidneyStone_SDOH (including RxNorm augmentation). |
| `scripts/ctakes_cli.py` | `python scripts/ctakes_cli.py` | Interactive runner to select pipeline/input, optionally mirror directories, and launch foreground/background runs (records metadata for run_status.py) |
| `scripts/run_status.py` | `python scripts/run_status.py --list` | Summarises recorded runs and reports progress using pipeline/async logs |

> **Runner logs:** `run_pipeline.sh` now emits `[pipeline][runner=N/M]` lines (threads, heap, start/finish) and honours `--background` to detach via `nohup`. `run_async.sh` mirrors the flag, shows per-shard progress/elapsed time, and writes shard logs under `<output>/.../shards/`.

## Validation workflows

| Scenario | Command | Outputs |
| --- | --- | --- |
| Optional sample smoke test | `bash scripts/validate_mimic.sh` | Requires local notes under `samples/mimic`; when present it writes to `outputs/validate_mimic/<pipeline>/` and maintains per-pipeline semantic manifest JSONs (concept CSVs, CUI counts, RxNorm rows, XMI mentions) |
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
| Compile + run | `bash scripts/build_dictionary.sh -- -u <umls> -o <dictionary>` | Passes arguments straight to `tools.dictionary.HeadlessDictionaryBuilder`; requires `CTAKES_HOME` and a UMLS key |

### KidneyStone_SDOH dictionary build

- Configuration lives in `resources/dictionary_configs/kidney_sdoh.conf`. Update `umls.dir` if your UMLS snapshot is stored elsewhere (the default points to `CtakesBun-bundle/umls_loader`).
- `bash scripts/build_dictionary_full.sh` normalises the UMLS snapshot layout, compiles the wrapper classes, runs discovery, and logs to `dictionaries/KidneyStone_SDOH/logs/build_<ts>.log`.
- Each run writes `dictionaries/KidneyStone_SDOH/merged_builder.properties` so the discovered SAB/TUI lists are pinned and the rebuilt XML/HSQL pair (and `_local` variant) land back inside the bundled cTAKES install.
- After the base dictionary is built, `tools.dictionary.DictionaryRxnormAugmenter` creates the `RXNORM` table in the HSQL store and patches the XML with the table reference.
- Discovery still widens the SAB list to everything in MRCONSO; comment out the `vocabularies=` override inside `scripts/build_dictionary_full.sh` if you prefer to keep only the curated subset from the config file.

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
|   |-- dictionary/HeadlessDictionary*.java
|   |-- assertion/DefaultSubjectAnnotator.java
|   |-- reporting/uima/*.java
|   |-- smoking/SmokingAggregateStep{1,2}Libsvm.java
|   `-- wsd/SimpleWsdDisambiguatorAnnotator.java
```

`build/` and `outputs/` are transient. If a run misbehaves, remove them (`rm -rf build outputs`) and rerun; the scripts will rebuild the Java helpers automatically.

## Troubleshooting

| Symptom | Quick fix |
| --- | --- |
| RegexSpanFinder warnings | Bundled bundle includes slimmed dash regex; update to this release to silence the message. |
| `java` missing or < 11 | Install Java 11+ (Debian/Ubuntu: `bash scripts/install_deps.sh` or `sudo apt-get install openjdk-17-jdk`) |
| `CTAKES_HOME` warnings | Export `CTAKES_HOME` or let the bundled distro load; rerun `scripts/flight_check.sh` to persist the value into `.ctakes_env` |
| Dictionary XML not found | Ensure `${CTAKES_HOME}/resources/org/apache/ctakes/dictionary/lookup/fast/KidneyStone_SDOH_local.xml` exists or pass `--dict <file>` |
| `validate.sh --limit` fails | Install `python3` / `python`, or run without `--limit` |
| Semantic groups look off | Update `resources/SemGroups.txt`; writers apply overrides via `SemGroupLoader` |

## Notes for future maintenance

| Area | Notes |
| --- | --- |
| Smoking pipeline | Local wrappers under `tools/smoking/` load the upstream `ctakes-smoking-status` aggregates |
| Drug pipeline | Depends on `tools.drug.DrugMentionAnnotatorWithTypes`; cTAKES bundle ships the patched descriptor so Piper can resolve the TypeSystem |
| Tool compilation | `scripts/run_pipeline.sh` recompiles Java helpers under `tools/` into `build/tools/` on every run |
| Release bundle | The shipped `CtakesBun-bundle` already includes `TsFastRelationSubPipe.piper`; rebuild the asset if you add new overrides so they land in `resources/` before cutting a release. |

That's it: a small toolkit that runs cTAKES pipelines, produces clean CSVs, and stays easy to follow.

### HTML visualization

Add `tools.reporting.uima.HtmlAnnotationOverviewWriter SubDirectory=html` to a pipeline (for example under the `// Writers` section). The writer emits `outputs/<run>/html/<docId>.html` with interactive toggles for core, WSD, drug, relation, and coreference layers. Use it when you want a quick visual diff of what each component contributes.

