Dictionary and WSD Guide
========================

Overview
--------
- Build a full UMLS rare-word dictionary (HSQL) equivalent to the cTAKES GUI DictionaryCreator.
- Run pipelines end-to-end with an offline dictionary (no UMLS prompts) and word-sense disambiguation (WSD).
- Optional: Build a local UMLS DB for YTEX WSD; provided a lightweight local WSD as default.

Key Outputs
-----------
- HSQL dictionary DB: `apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0/resources/.../FullClinical_AllTUIs/`
- Dictionary XML (offline copy created per run): `outputs/*/FullClinical_AllTUIs_*_local.xml`
- WSD outputs (smoke script):
  - XMI: `outputs/wsd_smoke/xmi/`
  - Tables: `outputs/wsd_smoke/{bsv_table,csv_table,html_table}/`

Build the Full Dictionary
-------------------------
- Script: `scripts/build_dictionary_full.sh`
  - Normalizes UMLS layout under `umls_loader/`.
  - Headless scan (SABs, languages, TUIs) and build.
  - Writes HSQL DB + `FullClinical_AllTUIs.xml` under cTAKES resources.
  - Creates an offline-local XML copy for headless runs.

Known gotcha: HSQL file URLs cannot include spaces. The run scripts copy the DB to `/tmp/ctakes_full/FullClinical_AllTUIs` and rewrite `jdbcUrl` accordingly.

Run WSD (local, default)
------------------------
- Pipeline: `pipelines/wsd/TsDefaultFastPipeline_WSD.piper`
  - Default tokenizer, chunker, dictionary lookup.
  - Local WSD AE: `tools.wsd.SimpleWsdDisambiguatorAnnotator` (no YTEX dependency).
  - Writers: XMI + Semantic tables (BSV, CSV, HTML).

- Script: `scripts/run_wsd_smoke.sh`
  - Creates an offline copy of the dictionary XML.
  - Ensures HSQL DB path is space-free and rewires `jdbcUrl`.
  - Compiles local tools.
  - Runs the WSD pipeline and writes outputs under `outputs/wsd_smoke/`.

YTEX WSD (optional)
-------------------
- Build a local UMLS DB with UMLS tables for YTEX:
  - Script: `scripts/build_ytex_umls_db.sh <META_OR_UMLS_DIR> <OUT_DB_BASE>`
    - Loads MRREL, MRSTY, MRCONSO (ENG) into HSQL (e.g., `/tmp/ctakes_ytex_umls/ytex_umls`).
    - Creates indexes.
    - Writes `apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0/resources/org/apache/ctakes/ytex/ytex.properties`.
  - Requirements: To use the original YTEX `SenseDisambiguatorAnnotator`, the environment also needs Hibernate 5 jars and an updated Spring config (not included by default). The included local WSD avoids this dependency.

CSV/BSV/HTML Tables
-------------------
- The WSD pipeline enables `SemanticTableFileWriter` in three styles:
  - BSV: default (pipe-delimited)
  - CSV: `TableType=CSV`
  - HTML: `TableType=HTML`
- See outputs under `outputs/wsd_smoke/` (or your selected `-o` directory).

Troubleshooting
---------------
- HSQL URL errors (spaces): The scripts copy the DB to `/tmp/...` and rewrite the `jdbcUrl`.
- UMLS prompt: Use the offline-local dictionary XML (the run scripts generate it automatically).
- Memory: Dictionary connect and WSD benefit from larger heaps (`-Xmx6g` to `-Xmx8g`).

Files and Scripts Added
-----------------------
- Tools:
  - `tools/wsd/SimpleWsdDisambiguatorAnnotator.java` – local WSD AE (single CUI selection).
  - `tools/ytex/LoadUmlsForYtex.java` – UMLS RRF → HSQL loader for optional YTEX DB.
- Pipelines:
  - `pipelines/wsd/TsDefaultFastPipeline_WSD.piper` – WSD + XMI + BSV/CSV/HTML tables.
- Scripts:
  - `scripts/run_wsd_smoke.sh` – run WSD pipeline and produce outputs.
  - `scripts/build_ytex_umls_db.sh` – build local UMLS DB for YTEX.

Future Improvements (Roadmap)
-----------------------------
- SAB-aware concepts in reports: When using a SAB-aware concept factory, include `SAB` and `CODE` columns alongside `CUI/TUI/PreferredText/CodingScheme` in the Clinical Concepts sheet.
- Graph-based WSD: Add a graph-relatedness WSD annotator backed by the local MRCONSO/MRSTY/MRREL DB (HSQL) for improved disambiguation, with Confidence reflecting graph scores.
