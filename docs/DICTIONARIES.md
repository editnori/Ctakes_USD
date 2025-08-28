# Dictionaries, Concept Factories, and WSD Scoring

This repo supports two dictionary/lookup styles and two ways to score concepts.

Key terms:
- HSQL Fast Dictionary: cTAKES “rare-word” index stored as HSQL files. Fast lookup of term → CUI/TUI.
- Concept Factory: The component that builds `refsem:UmlsConcept` features for a mention (CUI/TUI/preferredText/codingScheme/etc.).
- SAB (Source Vocabulary): UMLS source code (e.g., SNOMEDCT_US, RXNORM). Appears in UMLS MRCONSO.RRF as `SAB` and the source `CODE`.

## What We Use in WSD Smoke

The WSD smoke run uses a single dictionary per run — an offline copy of the FullClinical fast dictionary:

1) The runner copies your chosen dictionary XML to the run output dir and rewrites it for offline use:
   - `UmlsJdbcRareWordDictionary` → `JdbcRareWordDictionary` (uses HSQL files)
   - `UmlsJdbcConceptFactory` → `JdbcConceptFactory`
   - Repoints the `jdbcUrl` to a copy of the HSQL files under `/tmp/ctakes_full/` to avoid space-in-path issues.

2) This means only one dictionary is active: the HSQL fast dictionary referenced by the sanitized XML for the run. No other dictionaries are consulted during lookup.

Implications:
- Pros: fast, portable, no network/DB. Produces CUI/TUI/preferredText. `codingScheme` is the dictionary name (e.g., `FullClinical_AllTUIs`).
- Cons: not SAB-aware (no per-source `SAB`/`CODE`), no concept graph (MRREL), so advanced WSD measures based on semantic relatedness are unavailable.

## SAB-Aware Concept Handling

“SAB-aware” means the concept factory fills `codingScheme` with the true source vocabulary (e.g., `SNOMEDCT_US`) and also provides the source `code` (e.g., `22298006`). This requires reading MRCONSO.RRF (UMLS) at runtime (either via JDBC or a prebuilt DB). Options:

- UmlsJdbcConceptFactory (stock cTAKES):
  - Connects to a full UMLS RDBMS schema.
  - Emits per-source `codingScheme` (SAB) and `code`.
  - Heavyweight setup (RDBMS + full UMLS).

- YTEX-backed approach:
  - Uses a relational schema derived from UMLS tables (at minimum MRCONSO, MRSTY, MRREL).
  - Emits SAB/CODE and supports graph-based relatedness for WSD.
  - We include `tools/ytex/LoadUmlsForYtex.java` to build a lightweight HSQL DB with MRCONSO/MRSTY/MRREL from your UMLS `META` directory.

If you adopt a SAB-aware concept factory, the workbook can include `SAB` and `CODE` columns in addition to CUI/TUI.

## Scoring and Confidence

Fields used (existing in cTAKES):
- Mention-level: `textsem:IdentifiedAnnotation.confidence`.
- Concept-level: `refsem:UmlsConcept.score`.

Current WSD smoke scoring (custom, local):
- Implemented in `tools.wsd.SimpleWsdDisambiguatorAnnotator`.
- Picks one best concept per mention by token overlap between the sentence context and the candidate’s preferred text.
- ConceptScore = |context ∩ candidate| / |candidate|, tie‑break by longer preferred text.
- Sets: chosen concept `disambiguated=true`, `score=<0..1>`, and mention `confidence=<same 0..1>`.

Graph-based WSD (optional next step):
- Requires concept graph (MRREL) + MRCONSO to compute semantic relatedness among concepts in a window.
- YTEX provides several relatedness measures. You can build a filtered DB with our loader and implement a light WSD AE against HSQL to avoid external services.

## Which Dictionary is Used?

In our pipelines, exactly one dictionary is used per run: the one specified by the (sanitized) dictionary XML passed to Piper. The HSQL files behind it are the fast rare‑word index. Alternative setups (UMLS JDBC or YTEX) are options you can switch to; they are not active in the current WSD smoke pipeline.

## Report Columns and Sources

Workbook (`report.xml`) sheets:
- Overview: run timings and file paths (plus XMI document count) and high-level metrics.
- Modules: exact `load`/`add` lines from the `.piper` file.
- Clinical Concepts: chosen concept per mention (post-WSD) with fields consolidated: Section, Semantic Group/Type, Type, Polarity/Confidence/Assertion flags, CandidateCount/Disambiguated, CUI/TUI/PreferredText/CodingScheme/ConceptScore, Candidates, Text.
- CuiCounts/CuiList/Tokens: from unified writers.
- SheetGuide: column meanings and the source module.

Mapping:
- CUI/TUI/PreferredText: Dictionary + WSD
- CodingScheme: Dictionary concept factory (`FullClinical_AllTUIs` with fast dictionary; SAB if SAB-aware)
- ConceptScore/Confidence: Set by our WSD annotator (or by a graph-based WSD if enabled)
- Assertion flags (polarity/negated/uncertain/conditional/generic/subject/historyOf): Assertion modules

Notes:
- Some writers encode negation in CUI counts by prefixing a hyphen (e.g., `-C0000000`). The workbook’s CuiCounts sheet splits this into `CUI` and a `Negated` flag and strips the leading `-`.
- Some legacy cTAKES writers may append a "-c" suffix to CUIs in certain tables. The workbook normalizes CUIs by stripping a trailing "-c" when aggregating Mentions.
