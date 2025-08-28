Smoke Compare (Ts + WSD + Unified Writers)
==========================================

Purpose
-------
- Run the minimal set of thread-safe (Ts) pipelines that cover all full-pipeline module combinations in our catalog, with WSD inserted.
- Write consistent outputs per run: XMI, BSV table, and CSV table, under a single output root.

Catalog Coverage (8 combos)
---------------------------
- Sectioned variants (S_):
  - S_core: Sectioned core (tokenizer + POS + chunker + fast dictionary + WSD + assertion)
  - S_core_rel: S_core + Relations
  - S_core_temp: S_core + Temporal
  - S_core_temp_coref: S_core + Temporal + Coref
- Default variants (D_):
  - D_core_rel: Core + Relations + WSD
  - D_core_temp: Core + Temporal + WSD
  - D_core_temp_coref: Core + Temporal + Coref + WSD
  - D_core_coref: Core + Coref + WSD

S_core vs D_core (what’s the difference?)
----------------------------------------
- S_core (Sectioned core) uses the richer section-aware thread-safe tokenizer pipeline:
  - `TsFullTokenizerPipeline` → adds RegexSectionizer + Paragraph + List detection and sentence fixers before tokenization.
  - Better sentence/paragraph/section boundaries; populates the Section column with real section names (e.g., HISTORY OF PRESENT ILLNESS).
  - More robust for long clinical notes with headers, lists, and irregular line breaks.
  - Slightly heavier initialization than default; preferred for clinical documents where section context matters.
- D_core (Default core) uses the minimal thread-safe tokenizer pipeline:
  - `TsDefaultTokenizerPipeline` → simple segments + sentence detection + tokenization.
  - Section column typically shows `SIMPLE_SEGMENT` (no sectionizer), faster startup, fewer moving parts.
  - Good when you need speed and basic concept extraction, or for shorter/plaintext notes.

Which should I use?
-------------------
- Prefer S_core when: your notes have headings/sections/lists and you want sections in the report, more accurate sentence boundaries, and downstream AEs (relations/temporal/coref) benefit from better structure.
- Prefer D_core when: you want a leaner pipeline for quick runs or your input is already clean sentences without section headers.

Unification & Consistency
-------------------------
- Writers: all pipelines load `pipelines/includes/Writers_Xmi_Table.piper` so outputs/columns match.
- WSD/Assertion: all include `tools.wsd.SimpleWsdDisambiguatorAnnotator` and `TsAttributeCleartkSubPipe` (+ subject fix), with identical settings.
- Reporting: all runs produce the same multi-sheet workbook with color-coded headers and an “Interpretation Guide” embedded in the Pipeline Map.
- Temporal models: compare pipelines load `pipelines/includes/TsTemporalSubPipe_Fixed.piper` so classifier model paths resolve under `resources/org/apache/ctakes/temporal/models/...` shipped in the release.

Combinations Covered (8 total)
------------------------------
- Sectioned variants
  - S_core: Sectioned core (tokenizer+POS+chunker+fast dictionary+assertion) + WSD
  - S_core_rel: S_core + Relations
  - S_core_temp: S_core + Temporal
  - S_core_temp_coref: S_core + Temporal + Coref
- Default tokenizer variants
  - D_core_rel: Core + Relations + WSD
  - D_core_temp: Core + Temporal + WSD
  - D_core_temp_coref: Core + Temporal + Coref + WSD
  - D_core_coref: Core + Coref + WSD

Pipelines
---------
- See `pipelines/compare/*.piper`. Each one:
  - Loads Ts tokenizer/POSTagger/chunker
  - Loads TsDictionarySubPipe
  - Adds `tools.wsd.SimpleWsdDisambiguatorAnnotator`
  - Loads TsAttributeCleartkSubPipe
  - Adds combo-specific modules (TsRelationSubPipe, TsTemporalSubPipe, TsCorefSubPipe)
  - Loads `pipelines/includes/Writers_Xmi_Table.piper` (unified: XMI + tables + lists)

Writers and Metadata
--------------------
- Unified writers include: `pipelines/includes/Writers_Xmi_Table.piper`.
- Directory layout (per run):
  - `xmi/` — CAS XMI files
  - `bsv_table/` — BSV table files
  - `csv_table/` — CSV table files
  - `html_table/` — HTML table files
  - `cui_list/` — per-doc concepts list
  - `cui_count/` — per-doc CUI counts
  - `bsv_tokens/` — tokens + spans

Dictionary Handling
-------------------
- Uses the full clinical dictionary built under cTAKES resources.
- Per run, the script creates a local offline XML copy and rewrites the `jdbcUrl` to a space-free `/tmp/ctakes_full/<DICT_NAME>` location.

Run the Compare Smoke
---------------------
```
./scripts/run_compare_smoke.sh [SINGLE_NOTE_OR_DIR] [OUT_DIR]
```
- Default input: `samples/input/note1.txt`
- Default output: `outputs/compare/`

Optional Modules
----------------
- Smoking status: resources are present in cTAKES; we can add a dedicated pipeline variant (Core + SmokingStatus) once we confirm the specific AE classes to wire (lookup + classifier). Output writers will remain the same (BSV, CSV, XMI) to keep comparisons consistent.
