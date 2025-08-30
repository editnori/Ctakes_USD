# cTAKES 6.0.0 Catalog — Modules, Pipelines, and How To Run Them

**Author:** Layth M Qassem
**Date:** 2025-08-23
**Purpose:** One place that shows every NLP module available in this cTAKES build, every pipeline that ships in the jars, and how those pieces connect. I include a runnable stress‑test plan so we can evaluate speed and accuracy across all pipelines without guessing.

## How This Catalog Works

I scanned the distribution under `apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0/lib`. I pulled every embedded `.piper` and mapped which NLP modules each pipeline uses. The CSV is the authoritative index:

- Catalog CSV: `docs/ctakes_catalog.csv` (pipelines as rows, modules as columns; types: full, sub, example)

If we add or remove jars, we regenerate the CSV and this file stays truthful.

## Mental Model

Piper is a recipe. JARs are the engines. Sub‑pipelines are reusable blocks the full pipelines load. The fast dictionary gets its descriptor via `-l /path/to/dictionary.xml`. That’s the only input your pipeline needs beyond `-i` and `-o`.

## What’s In This Build (numbers that matter)

- 36 full pipelines (ready to run). 17 sub‑pipelines (building blocks). 24 example pipelines (demos). Source: the CSV.
- Clinical: DefaultFastPipeline, SectionedFastPipeline, and their thread‑safe Ts variants.
- Temporal: DefaultTemporalPipeline (+ relations), Sectioned variants, and Ts variants.
- Coreference: Temporal+Coref / Relation+Coref variants that load the coref sub‑pipe.
- Relation extractor: clinical modifier relations (degree, location, modifiers).
- Tokenizer stacks: DefaultTokenizerPipeline, FullTokenizerPipeline (section/paragraph/list aware), Ts variants.
- Dictionary: DictionarySubPipe (fast rare‑word dictionary). Assertion: AttributeCleartkSubPipe (negation, uncertainty, subject, etc.).

## Modules We Will Use (grouped)

Core text processing:
- Tokenization: `DefaultTokenizerPipeline` (simple) or `FullTokenizerPipeline` (section/paragraph/list aware).
- POS and chunking: `POSTagger` + `ChunkerSubPipe` to improve dictionary hits and relation features.

Entity extraction and attributes:
- Fast dictionary NER: `DictionarySubPipe` (pass `-l /path/to/dictionary.xml`).
- Assertion: `AttributeCleartkSubPipe` for negation, uncertainty, subject, history, conditional, generic.
 - Word‑sense disambiguation (WSD): `org.apache.ctakes.ytex.uima.annotators.SenseDisambiguatorAnnotator`. We insert this after dictionary lookup in every run so multi‑CUI mentions are resolved (sets `disambiguated=true` and prunes candidates). This jar ships in our build (`ctakes-ytex-uima-6.0.0.jar`).

Advanced semantics:
- Temporal: `TemporalSubPipe` for events, times, and temporal relations (event↔time, event↔event, event↔doc‑time).
- Coreference: `CorefSubPipe` to resolve pronouns and link mentions across sentences.
- Relations: `RelationSubPipe` for clinical modifier relations (degree, location, modifiers) outside of temporal.

Notes on other modules:
- Drug NER and Smoking Status ship as modules but do not include prebuilt `.piper` sub‑pipes in this distribution. If we need them, we add the AEs by class in a custom piper. The CSV reflects that (0s) because no shipping pipelines reference them.

If you want the source of truth, open the CSV and the `.piper` files—every mapping is backed by the jar contents.

## Pipelines We Will Test (and why)

Pick the pipeline that answers your question. Don’t guess.

- Entity extraction with attributes, fast: `org/apache/ctakes/clinical/pipeline/DefaultFastPipeline.piper`
- Same, but section/paragraph/list aware: `org/apache/ctakes/clinical/pipeline/SectionedFastPipeline.piper`
- Put events on a timeline: `org/apache/ctakes/temporal/pipeline/DefaultTemporalPipeline.piper`
- Resolve pronouns before you interpret events: `org/apache/ctakes/coreference/pipeline/DefaultTemporalCorefPipeline.piper`
- Add degree/location/modifier relations explicitly: `org/apache/ctakes/relation/extractor/pipeline/DefaultRelationPipeline.piper` (or the Sectioned/Ts variants)

Our test set is “all full pipelines” in `docs/ctakes_catalog.csv` (36 runs). That covers clinical, temporal, relation, and coref variants, including thread‑safe `Ts*` forms. At a glance:

Clinical (ctakes-clinical-pipeline):
- DefaultFastPipeline, SectionedFastPipeline
- TsDefaultFastPipeline, TsSectionedFastPipeline

Temporal (ctakes-temporal):
- DefaultTemporalPipeline, SectionedTemporalPipeline
- DefaultRelationTemporalPipeline, SectionedRelationTemporalPipeline
- TsDefaultTemporalPipeline, TsSectionedTemporalPipeline
- TsDefaultRelationTemporalPipeline, TsSectionedRelationTemporalPipeline

Coreference (ctakes-coreference):
- DefaultTemporalCorefPipeline, SectionedTemporalCorefPipeline
- DefaultCorefPipeline, DefaultAdvancedPipeline, DefaultRelationCorefPipeline
- SectionedCorefPipeline, SectionedAdvancedPipeline, SectionedRelationCorefPipeline
- TsDefaultTemporalCorefPipeline, TsSectionedTemporalCorefPipeline
- TsDefaultCorefPipeline, TsDefaultAdvancedPipeline, TsDefaultRelationCorefPipeline
- TsSectionedCorefPipeline, TsSectionedAdvancedPipeline, TsSectionedRelationCorefPipeline

Relation extractor (ctakes-relation-extractor):
- DefaultRelationPipeline, SectionedRelationPipeline
- TsDefaultRelationPipeline, TsSectionedRelationPipeline

Thread‑safe Ts variants use the same components and optionally set `threads N` in the piper.

### What “Sectioned” Means

Sectioned pipelines use `FullTokenizerPipeline` up front. That adds a regex sectionizer, paragraph boundaries, and list/table handling before sentence detection and tokenization. Use these when notes have clear headers (e.g., "MEDICATIONS:") and bullet lists. Otherwise, the default tokenizer is fine.

## Run Commands (copy/paste)

Set four variables once. Then change only `-p` to walk pipelines.

```
export CTAKES_HOME="$(pwd)/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
export IN="/path/to/input_txt_dir"
export OUT="/path/to/output_dir"
export DICT_XML="/path/to/dictionary.xml"   # required by fast dictionary pipelines
```

Default Fast (entities + assertion):
```
java -Xms2g -Xmx6g \
  -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/lib/*" \
  org.apache.ctakes.core.pipeline.PiperFileRunner \
  -p org/apache/ctakes/clinical/pipeline/DefaultFastPipeline.piper \
  -i "$IN" -o "$OUT" -l "$DICT_XML"
```

Temporal (events + times + temporal relations):
```
java -Xms2g -Xmx6g \
  -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/lib/*" \
  org.apache.ctakes.core.pipeline.PiperFileRunner \
  -p org/apache/ctakes/temporal/pipeline/DefaultTemporalPipeline.piper \
  -i "$IN" -o "$OUT" -l "$DICT_XML"
```

Temporal + Coref (resolve pronouns first):
```
java -Xms2g -Xmx6g \
  -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/lib/*" \
  org.apache.ctakes.core.pipeline.PiperFileRunner \
  -p org/apache/ctakes/coreference/pipeline/DefaultTemporalCorefPipeline.piper \
  -i "$IN" -o "$OUT" -l "$DICT_XML"
```

Sectioned variants: replace `Default*` with `Sectioned*` in `-p`.
Thread‑safe variants: use the `Ts*` pipeline under the same package.

## Complete Stress Test Configuration (what we run, what we set)

We are benchmarking all “full” pipelines (Ts preferred) with a single, consistent config so speed and accuracy comparisons are meaningful.

What we set (consistent across runs):
- Java heap: `-Xms2g -Xmx6g` (bump to 8–12 GB only if GC or OOM shows up on long notes).
- GC: `-XX:+UseG1GC` (stable choice for mixed workloads).
- Dictionary: pass `-l "$DICT_XML"` whenever `needs_dict=1` in `docs/stress_test_plan.csv`.
- XMI output: add `--xmiOut "$OUT/xmi"` if you want XMI for every run (useful for diffing). Not all shipped pipelines add an XMI writer explicitly; the runner option forces XMI output consistently.
- Threads: prefer `Ts*` pipelines. Leave their internal `threads N` as declared (mostly `threads 3`). If you need a global thread target, copy the Ts piper and set `threads N` at the top; keep it the same across runs.
- WSD enabled for all runs: after `DictionarySubPipe`, insert `add org.apache.ctakes.ytex.uima.annotators.SenseDisambiguatorAnnotator` in each pipeline we execute. (We maintain local WSD‑enabled copies of the shipped `.piper` files for stress runs.)

What we do not set (and why):
- JDBC writers (I2B2, generic JDBC), FHIR writers: not part of the speed/accuracy question; they introduce external I/O variance.
- Per‑AE model parameters (e.g., temporal relation model paths): default models shipped in jars are used; changes would make comparisons across pipelines invalid.
- UMLS API key (`--key`): only required if your dictionary/pipeline enforces it. Our runs assume local dictionary descriptor and DB/BSV resources.
- Example pipeline settings (minimumSpan, PBJ demo toggles): they are demo‑specific, not relevant to production pipelines.

### Outputs beyond XMI (CSV/TSV/pretty)

If you want table/CSV‑like outputs in addition to XMI, add these CAS Consumers to your pipeline:
- `org.apache.ctakes.core.cc.SemanticTableFileWriter` — table of recognized semantic items (TSV by default).
- `org.apache.ctakes.core.cc.TokenTableFileWriter` — tokens table (useful for audits).
- `org.apache.ctakes.core.cc.CuiListFileWriter` / `CuiCountFileWriter` — per‑doc CUI list/counts.
- `org.apache.ctakes.core.cc.pretty.plaintext.PrettyTextWriter` — human‑readable annotated plaintext.
- `org.apache.ctakes.core.cc.XmiWriterCasConsumerCtakes` or `org.apache.ctakes.core.cc.FileTreeXmiWriter` — XMI outputs.

Piper snippet (add at the end):
```
add org.apache.ctakes.core.cc.SemanticTableFileWriter
add org.apache.ctakes.core.cc.XmiWriterCasConsumerCtakes
```
Note: table writers emit TSV by default; convert to CSV if needed. JDBC/I2B2 writers exist but introduce external I/O variance; we skip them for stress tests.

## Dictionaries: Build, Wire, and Evaluate (Kidney Stone registry first)

We treat the dictionary as a first‑class artifact. Pipelines that load `DictionarySubPipe` rely on your `dictionary.xml` passed via `-l`. We build a comprehensive dictionary first, benchmark it, then trim without losing accuracy.

How pipelines use the dictionary (runtime):
- `DictionarySubPipe` declares `cli LookupXml=l`, so `-l /path/to/dictionary.xml` injects it into `DefaultJCasTermAnnotator`.
- In parallel runs (your shard script), write a per‑runner copy of the HSQL DB and rewrite `jdbcUrl` to a `/dev/shm/...` path to avoid file locks.

Build options (UMLS → BSV/HSQL/Lucene):
- GUI (interactive): `org.apache.ctakes.gui.dictionary.DictionaryCreator`.
- Headless (repeatable): `java -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/lib/*:<repo>/dictionaries/FullClinical_AllTUIs/build${HSQLJAR:+:$HSQLJAR}" tools.HeadlessDictionaryBuilder -p docs/builder_full_clinical.properties`.
  - Requires HSQLDB 1.8.x driver (`org.hsqldb.jdbcDriver`). Set `HSQLJAR=/path/to/hsqldb-1.8.0.10.jar`.

Headless builder.properties (copy/paste template)
See docs/builder_full_clinical.properties for a ready-to-run "FullClinical_AllTUIs" config (ENG, broad SABs, ALL TUIs, both BSV and HSQL outputs). Adjust umls.dir and output.dir.

RRF files (what the builder uses)
- Required: `MRCONSO.RRF` (terms), `MRSTY.RRF` (semantic types), `MRSAB.RRF` (sources/SABs).
- Recommended: `MRRANK.RRF` (term type ranking to prefer better strings), `MRXW_ENG.RRF` (English word index; helps normalization).
- Optional: `MRDEF.RRF` (definitions), `MRREL.RRF`/`MRHIER.RRF` (relations/hierarchy), `MRMAP.RRF`/`MRSMAP.RRF` (mappings). These are not required for fast dictionary lookup but can support advanced workflows.

Store choices (inside dictionary.xml)
- BSV (`BsvRareWordDictionary`): simplest & fast. Great default.
- HSQL (`JdbcRareWordDictionary`): DB‑backed; use per‑runner DBs for parallelism.
- Lucene (`LuceneRareWordDictionary`): only if you need search/fuzzy behavior; not needed for standard fast pipelines.

Full build → Trim loop
- Build KidneyStone_Full (broad SAB/TTY/TUI) and run the stress plan.
- Analyze TSV/XMI: keep SABs/TTYs/TUIs that contribute; drop low/no‑yield ones and rebuild a “KidneyStone_Slim”.
- Re‑benchmark. Adopt the smallest dictionary that preserves your outcome metrics.

Benchmark signals to log
- Throughput: elapsed, docs/min, CPU, maxrss.
- Coverage: unique CUIs, TUIs, SAB distribution.
- WSD: % disambiguated=true; avg concepts per mention.
- Clinical: negation rate, subject=patient rate, DocTimeRel distribution for events.

### Dictionary Components & Parameters (Full build)

Use `docs/dictionary_components_full.csv` as the authoritative catalog of components and parameters for the Full build. It covers:
- Builder inputs: paths, languages, SAB list, TTY/TUI policy (ALL for Full), normalization/filters, output stores (BSV/HSQL/Lucene).
- RRF dependencies: which RRFs are required/recommended/optional and why.
- Dictionary XML: backend selection (Bsv/Jdbc/Lucene), bsvPath/jdbcUrl/jdbcDriver/caseSensitive, consumer binding.
- Pipeline wiring: `-l` alias, per‑runner HSQL rewrite, WSD annotator placement.

Rationale (why we set these defaults for Full):
- We do not prefilter TUIs/TTYs for Full — we want maximum recall across admission, discharge, ED, inpatient, outpatient, and radiology notes, including SDOH concepts. We constrain only by language (ENG) and by an intentionally broad clinical SAB set. We will trim after we see real usage.
- Normalization is on and suppressible/obsolete are off to reduce noise, without sacrificing coverage of valid clinical concepts.
- BSV is our baseline store for simplicity and speed; HSQL is also emitted to benchmark DB‑backed lookup and to support very large dictionaries. Lucene is left off unless we identify a concrete retrieval use case.

## Build & Test Commands (FullClinical_AllTUIs in BSV and HSQL)

Build (headless):
```
export CTAKES_HOME="$(pwd)/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
java -Xms2g -Xmx6g \
  -cp "$CTAKES_HOME/lib/*" \
  org.apache.ctakes.gui.dictionary.DictionaryBuilder \
  -p docs/builder_full_clinical.properties
```

Outputs (under the output.dir in the properties):
- terms.bsv (for BSV store)
- hsqldb/ (folder with .script/.properties files for HSQL store)
- dictionary.xml (points to bsvPath or jdbcUrl; we keep both entries and choose at runtime)

Quick sanity checks:
```
# Count BSV rows
wc -l "$(grep -o 'output.dir=.*' docs/builder_full_clinical.properties | cut -d= -f2)/terms.bsv"

# Inspect dictionary.xml to confirm bsvPath and jdbcUrl entries
sed -n '1,200p' "$(grep -o 'output.dir=.*' docs/builder_full_clinical.properties | cut -d= -f2)/dictionary.xml"
```

Test run (DefaultFastPipeline with WSD):
```
IN="/path/to/input_txt_dir"
OUT="/path/to/out"
DICT_XML="$(grep -o 'output.dir=.*' docs/builder_full_clinical.properties | cut -d= -f2)/dictionary.xml"

# Use BSV (dictionary.xml has bsvPath set) — pass -l
java -Xms2g -Xmx6g \
  -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/lib/*" \
  org.apache.ctakes.core.pipeline.PiperFileRunner \
  -p org/apache/ctakes/clinical/pipeline/DefaultFastPipeline.piper \
  -i "$IN" -o "$OUT" -l "$DICT_XML"
```

Parallel HSQL (per‑runner rewrite):
```
RUNNER_XML="$OUT/runner_001.xml"
cp "$DICT_XML" "$RUNNER_XML"
sed -i "s|jdbc:hsqldb:file:[^"]*|jdbc:hsqldb:file:/dev/shm/hsqldb_runner_001|g" "$RUNNER_XML"
# then run -l "$RUNNER_XML" in that runner
```

Note: we insert WSD in our local pipeline copies: add
`add org.apache.ctakes.ytex.uima.annotators.SenseDisambiguatorAnnotator`
right after `load DictionarySubPipe` so multi‑CUI spans are disambiguated.

## Run Order (clean and reproducible)

1) Build the dictionary (relative paths, logs):
- `./scripts/build_dictionary_full.sh`

2) Quick smoke test (single pipeline):
- `./scripts/test_dictionary_full.sh -i ./samples/input -o ./outputs/test_full`

3) Cluster/parallel run (if you want sharded throughput now):
- Export `IN=./path/to/input`
- Optionally export `PIPER=./pipelines/wsd/TsDefaultFastPipeline_WSD.piper`
- `./scripts/ctakes_cluster_run.sh`

4) Stress plan (all full pipelines; WSD variants referenced directly in CSV):
- `./scripts/run_stress_plan.sh ./dictionaries/FullClinical_AllTUIs/dictionary.xml ./path/to/input ./outputs/stress_full`

We do not resolve pipelines at runtime: docs/stress_test_plan.csv points directly to local WSD pipelines for the families we care about (DefaultFast, SectionedFast, Temporal, Temporal+Coref, Ts variants). This keeps the plan explicit and reproducible.

## Output Schema Reference (XMI and TSV/CSV)

Core CAS fields (any annotation)
- begin, end: character offsets in source text.
- coveredText: original substring at [begin, end).
- type: annotation class (e.g., DiseaseDisorderMention, SignSymptomMention, MedicationMention, EventMention, TimeMention, Sentence, Token).

Concept fields (DictionarySubPipe → UmlsConcept in ontologyConceptArr)
- cui: UMLS CUI (e.g., C0018799). Global concept identifier.
- tui: UMLS semantic type (e.g., T184). Group by TUI to aggregate categories.
- codingScheme: vocabulary (SNOMEDCT_US, RXNORM, LOINC, …).
- conceptCode: code in that vocabulary (e.g., SNOMED code).
- preferredText: human‑readable preferred term.
- disambiguated: true/false. With WSD enabled, expect one concept and true.

Assertion fields (AttributeCleartkSubPipe → on IdentifiedAnnotation)
- polarity: 1 = affirmed/present, −1 = negated.
- uncertainty: 1 = uncertain/possible, 0 = certain.
- conditional: 1 = hypothetical/conditional, 0 = not conditional.
- generic: 1 = generic/non‑specific, 0 = specific.
- subject: typically “patient” vs. non‑patient (e.g., “family_member”).
- historyOf: 1 = past history, 0 = not marked as history.
- discoveryTechnique: integer code for discovery method (dictionary, ML, …). Use for audits; mapping can vary by version.
- confidence: float, often 0.0 unless explicitly set by a component.

Temporal fields (TemporalSubPipe)
- DocTimeRel on EventMention: BEFORE, OVERLAP, AFTER, VAGUE.
- Relations: event–time and event–doc links (as relation annotations in XMI).
- TimeMention normalization: sometimes present depending on models; otherwise treat coveredText as the time string.

SemanticTableFileWriter (TSV) — typical columns
- Inspect the first TSV header row; treat it as authoritative for this build. Common columns include:
  - begin, end, coveredText, type
  - cui, tui, codingScheme, conceptCode, preferredText
  - polarity, uncertainty, conditional, generic, subject, historyOf
  - (sometimes) DocTimeRel when rows correspond to events

Interpretation:
- polarity = −1 → negate clinically.
- subject ≠ patient → downrank or exclude for patient‑level outcomes.
- uncertainty/conditional = 1 → candidate/possible; don’t treat as certain without business logic.
- With WSD on, most mentions have one concept. If not, log and inspect.

Pointer: `docs/ctakes_runtime_params.csv` lists the knobs we found (JVM, runner, piper aliases, `set` directives, `threads`, and model paths) with where/how to set them.

## What “Sectioned” Means

Sectioned pipelines use `FullTokenizerPipeline` up front. That adds a regex sectionizer, paragraph boundaries, and list/table handling before sentence detection and tokenization. Use these when notes have clear headers (e.g., “MEDICATIONS:”) and bullet lists. Otherwise, the default tokenizer is fine.
## Stress‑Test Plan (don’t miss anything)

Use `docs/stress_test_plan.csv`. It’s ordered to start with our default clinical run, then we tack on capabilities (temporal, coref, relations), Ts preferred, example pipelines excluded. Each row includes: modules present, whether it needs a dictionary, and what changed vs. the previous run (`added_modules`).

Columns that matter:
- `modules_list`: which modules this pipeline uses (tokenizer, POS, chunker, fast_dictionary, assertion, temporal, coreference, relation_extractor, …).
- `added_modules`: deltas vs. the prior run so we see what we’re tacking on (e.g., temporal, coreference).

### XMI fields to verify while we benchmark
- IdentifiedAnnotation: `polarity` (negation), `uncertainty`, `conditional`, `generic`, `subject`, `historyOf`, `discoveryTechnique`, `confidence` (often 0.0 unless a component sets it).
- Concepts: `ontologyConceptArr` with `cui`, `tui`, `codingScheme`, `conceptCode`, `preferredText`. With WSD on, `disambiguated=true` and array length should be 1 for most mentions.
- Temporal: `DocTimeRel` on events (BEFORE/OVERLAP/AFTER/VAGUE), Event–Time/DocTime links.

Bash (POSIX) version:
```
CSV="docs/ctakes_catalog.csv"
while IFS=, read -r Pipeline Type Jar Path rest; do
  [ "$Type" != "full" ] && continue
  p="$Path"
  name=$(basename "$p" .piper)
  outdir="$OUT/$name"
  mkdir -p "$outdir"
  # detect if this pipeline expects fast dictionary
  needs_dict=$(awk -F, -v P="$p" 'NR==1{for(i=1;i<=NF;i++){h[$i]=i}} $4==P{print $h["fast_dictionary"]}' "$CSV")
  if [ "${needs_dict:-0}" = "1" ]; then
    dict_args=( -l "$DICT_XML" )
  else
    dict_args=()
  fi
  echo "==> $name"
  /usr/bin/time -f '%E user=%U sys=%S maxrss=%M' \
  java -Xms2g -Xmx6g -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/lib/*" \
    org.apache.ctakes.core.pipeline.PiperFileRunner \
    -p "$p" -i "$IN" -o "$outdir" "${dict_args[@]}" || true
done < <(tail -n +2 "$CSV")
```

I keep the output directory per pipeline so we can diff XMI counts and file sizes. I log `time` stats to get elapsed, CPU, and memory. If you want a CSV of results, wrap the loop and append per‑pipeline metrics (pipeline name, elapsed seconds, doc count, XMI count, average sec/doc).

## How Sub‑Pipelines Relate To Full Pipelines

Sub‑pipelines are building blocks: tokenizer, chunker, dictionary, assertion, temporal, coreference, relations. Full pipelines `load` sub‑pipelines in the right order. If you need to change behavior, swap one sub‑pipeline at a time and rerun the same corpus. The CSV marks `Type=sub` so you can see what exists before you edit anything.

## Ground Truth

I do not rely on memory. The CSV is generated from the jars we ship. Pipelines come from these packages:

- Clinical full pipelines: `ctakes-clinical-pipeline-6.0.0.jar:/org/apache/ctakes/clinical/pipeline/*.piper`
- Temporal: `ctakes-temporal-6.0.0.jar:/org/apache/ctakes/temporal/pipeline/*.piper`
- Coreference: `ctakes-coreference-6.0.0.jar:/org/apache/ctakes/coreference/pipeline/*.piper`
- Relation extractor: `ctakes-relation-extractor-6.0.0.jar:/org/apache/ctakes/relation/extractor/pipeline/*.piper`
- Tokenizers: `ctakes-core-6.0.0.jar:/org/apache/ctakes/core/pipeline/*Tokenizer*.piper`
- Dictionary sub‑pipe: `ctakes-dictionary-lookup-fast-6.0.0.jar:/org/apache/ctakes/dictionary/lookup/fast/pipeline/*.piper`
- Assertion: `ctakes-assertion-6.0.0.jar:/org/apache/ctakes/assertion/pipeline/*.piper`
- Demos/examples: `ctakes-examples-6.0.0.jar:/org/apache/ctakes/examples/pipeline/*.piper`

If a pipeline changes upstream, we rerun the scan, regenerate the CSV, and this catalog stays accurate.

## Confidence, Scoring, and Word Sense Disambiguation (WSD)

Two direct questions I get when teams come from MetaMap or CUI matchers with explicit confidence:

- Concept confidence: The fast dictionary lookup in cTAKES is a rules/matching engine. It does not emit a MetaMap‑style numeric confidence for each CUI by default. Assertion and temporal components are machine‑learned (ClearTK), but the shipping pipelines annotate categorical properties (e.g., polarity/negation, uncertainty, DocTimeRel) rather than exposing classifier probabilities. If you need scores, you’d instrument those AEs or add writer code to emit their internal margins/probabilities. Not part of default outputs.

- Word Sense Disambiguation (WSD): cTAKES 6.0.0 does not ship a dedicated WSD pipeline or annotator. Disambiguation is handled indirectly by dictionary design (rare‑word dictionary, windowing, chunk boundaries) and by context modules (assertion, temporal, coref) that filter or classify mentions. If you need WSD proper, you integrate an external WSD component or add a custom AE.


