# Excel Report (Multi-Sheet)

Generates a single Excel-compatible workbook (XML) per run with multiple sheets:
- Overview: Run Info + high-level metrics plus “Top Concepts (Chosen by WSD)”.
- Pipeline Map: pipeline order (with friendly labels) and a mapping of “Clinical Concepts” columns to their source modules.
- Processing Metrics: basic per-AE activity from the run log + counts of files written by writers.
- Clinical Concepts: consolidated per-mention details across all docs (includes Section, Semantic Group/Type, Confidence, ConceptScore, and Candidates).
- CuiCounts: aggregated CUI counts per doc (with Negated column).
- Tokens: aggregated tokens and spans.

Why XML, not .xlsx?
- This uses the Excel 2003 XML (SpreadsheetML) format which Excel opens natively. It avoids pulling in extra libraries.
- You can open the `.xml` in Excel and “Save As” `.xlsx` if desired.

Build a report for a run:

```
# scripts/run_wsd_smoke.sh now auto-builds the workbook as part of the run.
# Disable with --no-report if needed.

# Manual (re)build:
./scripts/build_xlsx_report.sh -o outputs/wsd_smoke
# Workbook name defaults to: ctakes-report-<run>-<pipeline>-<dictionary>-<timestamp>.xml
# Example: outputs/wsd_smoke/ctakes-report-wsd_smoke-TsDefaultFastPipeline_WSD-FullClinical_AllTUIs_wsd_local-YYYYMMDD-HHMMSS.xml
```

Optional arguments:
- `-p <pipeline.piper>`: pipeline file to list modules (auto-discovered from log if available)
- `-l <run.log>`: run log path (auto-discovered under `logs/` or `run.log`)
- `-d <dict.xml>`: dictionary XML path (auto-detected in output dir)
- `-w <workbook.xml>`: output workbook path

Interpreting the sheets:
- See `Pipeline Map` for pipeline order and which module produced each Clinical Concepts column.
- `Clinical Concepts` rows reflect the chosen concept (post-WSD). All candidates are preserved in XMI; best is at index 0 and marked `disambiguated=true`.
- Section names will be `SIMPLE_SEGMENT` unless a sectionizer is used (compare pipelines include one in sectioned variants).
- `Confidence` (in Clinical Concepts) reflects mention-level confidence when present. The bundled simple WSD also sets
  mention confidence and the chosen concept's `score` based on a normalized label-context token overlap.
- `Candidates` (in Clinical Concepts) lists all candidate concepts for the mention as `CUI:TUI:PreferredText; …` with the chosen concept first.

Visual cues:
- Header colors indicate sources:
  - Dictionary (CUI/TUI/PreferredText/CodingScheme/Semantic Type/Group): yellow
  - WSD (Confidence/ConceptScore/Disambiguated/Candidates/CandidateCount): light blue
  - Assertion (Polarity/Negated/Uncertain/Conditional/Generic/Subject/History): light red
  - Tokenization/Text/Begin/End/Document: light green
  - Meta (Guide/Modules/Overview): violet

Assertion/Boolean fields (meanings + examples):
- Polarity: integer assertion value; `-1` means negated, `1` means affirmed (present). Example: “no chest pain” → Polarity `-1`.
- Negated: boolean derived from Polarity; `true` if Polarity < 0, else `false`. Example: “no fever” → Negated `true`.
- Uncertain: boolean for uncertainty in the statement. Example: “possible pneumonia” → Uncertain `true`.
- Conditional: boolean for conditional mentions. Example: “if pain worsens, take ibuprofen” → Conditional `true`.
- Generic: boolean for non–patient-specific/generalized statements. Example: “aspirin can cause bleeding” → Generic `true`.
- Subject: the person/entity; typically `patient`, can be `family_member`, etc. Example: “family history of diabetes” → Subject `family_member`.
- HistoryOf: integer (usually `0` or `1`); `1` signals a historical or family-history context. Example: “history of MI” → HistoryOf `1`.

Keeping writers consistent:
- All pipelines load the same writer include: `pipelines/includes/Writers_Xmi_Table.piper`, so directory layout is the same across runs.

Cluster runs and consolidation:
- The cluster runner consolidates `shard_*` outputs into top-level folders (xmi, bsv_table, csv_table, html_table, cui_list, cui_count) before building reports. If running the report tool manually on a sharded run, it can also aggregate across `shard_*` folders when top-level folders are absent.
\n+Future improvements
-------------------
- Add `SAB` and source `CODE` columns when a SAB-aware concept factory is used (UMLS JDBC or YTEX-backed).
- Integrate graph-based WSD scoring (semantic relatedness) and surface the score in `Confidence/ConceptScore`.
