# cTAKES USD Clean Toolkit

This repository trims the original Ctakes_USD project down to the essentials and keeps it aligned with the bundled apache cTAKES 6.x distribution:

- **Dictionary builder** ? `tools/HeadlessDictionary{Creator,Builder}.java` plus `scripts/build_dictionary.sh` to compile and launch the headless UMLS dictionary builder.
- **Focused pipelines** ? four Piper files (`core`, `sectioned`, `smoke`, `drug`) with a shared OPTIONAL_MODULES hook so we can toggle temporal/coref support without duplicating pipelines.
- **Lean writers** ? pipelines now emit only CAS XMI, a per-document concepts CSV, and CUI counts. The drug pipeline also records RxNorm rows.
- **Run tooling** ? `scripts/run_pipeline.sh` adds autoscale heuristics (threads + heap), optional temporal/coref modules, and honours extra Java options.
- **Async runner** ? `scripts/run_async.sh` shards an input directory across multiple `run_pipeline.sh` workers, autoscaling shards/threads/heap and consolidating outputs plus summary CSVs.
- **Validation helpers** ? `scripts/validate.sh` for ad-hoc sampling and `scripts/validate_mimic.sh` for the 100-note smoke set.
- **Flight checks** ? `scripts/flight_check.sh` validates Java/CTAKES_HOME, pipeline presence, and performs a dry-run check.

Everything else?compare clusters, giant report builders, archived outputs?has been removed so the repo stays clean. The distributable cTAKES bundle lives under `CtakesBun-bundle/` and is kept free of generated outputs.

## Prerequisites

- Java 11 or newer on PATH (`java -version`).
- cTAKES 6.x installation. If `CTAKES_HOME` is unset, the scripts fall back to the bundled copy at `CtakesBun-bundle/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0`.
- Bash shell (Git Bash on Windows is fine).

Run the flight check at any time:

```bash
bash scripts/flight_check.sh
```

## Dictionary builder

```bash
# Compile classes only (drops .class files in build/dictionary/)
bash scripts/build_dictionary.sh --compile-only

# Compile + run (everything after "--" is passed straight to HeadlessDictionaryBuilder)
bash scripts/build_dictionary.sh -- -u /path/to/UMLS -o /path/to/dictionary
```

`BUILD_DIR=/custom/path` overrides the default output directory. The script automatically wires `${CTAKES_HOME}/lib/*` onto the classpath.

## Pipelines

| Key | Piper file | Purpose |
| --- | --- | --- |
| `core` | `pipelines/core/core_wsd.piper` | Default fast pipeline + dictionary lookup + WSD. |
| `sectioned` | `pipelines/sectioned/sectioned_core_wsd.piper` | Section-aware pipeline with relations. |
| `smoke` | `pipelines/smoke/sectioned_smoke_status.piper` | Sectioned pipeline with smoking-status annotators. |
| `drug` | `pipelines/drug/drug_ner_wsd.piper` | Sectioned pipeline with ctakes-drug-ner (adds RxNorm CSVs). |

All four pipelines call `tools.fixes.DefaultSubjectAnnotator` to ensure assertion subjects never land as `null` (cTAKES writers can crash otherwise), then write:

- `xmi/` ? CAS snapshots per note.
- `concepts/` ? Per-document concept CSVs written by `SimpleConceptCsvWriter`.
- `cui_count/` ? Frequency counts (cTAKES `CuiCountFileWriter`).
- `rxnorm/` ? Only for the `drug` pipeline via `DrugRxNormCsvWriter`.

### Run a pipeline

```bash
bash scripts/run_pipeline.sh \
  --pipeline sectioned \
  --with-temporal \
  --with-coref \
  --autoscale \
  -i /data/notes \
  -o /runs/sectioned_temporal
```

Highlights:

- `--autoscale` inspects CPU + RAM and applies sensible defaults (threads = cores/2, heap ? 60% of RAM capped between 2?24 GB).
- `--threads <N>` overrides the `threads` directive inside the Piper (the script rewrites a temporary copy).
- `--xmx <MB>` / `--java-opts "..."` extend `CTAKES_JAVA_OPTS` before launching `PiperFileRunner`.
- `--dry-run` prints the computed command.

If `CTAKES_HOME` is unset and the bundled cTAKES exists, the script uses it automatically.

### Run asynchronously

```bash
bash scripts/run_async.sh \
  --pipeline smoke \
  --autoscale \
  -i /data/notes \
  -o /runs/smoke_async
```

`run_async.sh` shards the input directory, launches `run_pipeline.sh` for each shard (in parallel), then consolidates shard outputs into:

- `xmi/`, `concepts/`, `cui_count/`, and optionally `rxnorm/` under `<output>/<pipeline>/<timestamp>/`.
- `concepts_summary.csv` (and `rxnorm_summary.csv` when applicable) built by concatenating per-document CSVs.

Use `--dry-run` to view the planned per-shard commands without executing them. Manual overrides (`--shards`, `--threads`, `--xmx`, `--dict`, etc.) pass straight through to the child runs.

## Validation workflows

General sampler:

```bash
bash scripts/validate.sh \
  --pipeline smoke \
  --limit 20 \
  -i /data/notes \
  -o /runs/validate_smoke
```

This copies the first 20 `.txt/.xmi/.xml` files into a temp folder (requires `python3` or `python`) before running.

Bundled 100-note smoke test:

```bash
bash scripts/validate_mimic.sh
```

`--with-temporal`, `--with-coref`, and `--dry-run` pass straight through to `scripts/validate.sh`.

## Repository layout

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
|   |-- smoking/*.java
|   `-- wsd/SimpleWsdDisambiguatorAnnotator.java
`-- resources_override/org/apache/ctakes/drugner/ae/DrugMentionAnnotator_WithTypes.xml
```

## Troubleshooting

- `CTAKES_HOME`: export it if you do not want to rely on the bundled distribution.
- `java.lang.NoClassDefFoundError`: ensure Java 11+ is active and `${CTAKES_HOME}/lib` holds cTAKES jars.
- `python not found` when using `validate.sh --limit`: install Python (`python3` or `python`) or drop `--limit`.
- Run `bash scripts/flight_check.sh` after any environment change; it reports missing prerequisites, warns about absent sample notes, and verifies `run_pipeline.sh` with `--dry-run`.

That's the cleaned setup?dictionary tooling, four lean pipelines, autoscale-aware single-run and async runners, flight checks, and a predictable validation story.
