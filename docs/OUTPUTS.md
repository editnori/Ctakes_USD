# Outputs Overview

This repo configures consistent outputs across all pipelines (smoke and compare).
Each run writes the same directories under the chosen output path.

Directories:

- `xmi/`: Full CAS serialized to XMI per document.
- `bsv_table/`: Semantic table (BSV) of clinical mentions per document.
- `csv_table/`: Same table in CSV.
- `html_table/`: Same table in HTML for quick viewing.
- `cui_list/`: Concept list (basic) per document.
- `cui_count/`: Counts of CUIs per document.
- `bsv_tokens/`: Token list with text spans per document.
 - `report.xml`: Excel 2003 XML workbook (multi-sheet) summarizing the run.

File naming pattern: `<docname>_table.(BSV|CSV|HTML)` for tables, and `<docname>.txt.xmi` for XMI.

Table columns (BSV/CSV/HTML):

- Semantic Group, Semantic Type, Section, Span, Negated, Uncertain, Generic,
  CUI, Preferred Text, Document Text

Notes:

- The table shows the single chosen concept per mention. If Word Sense
  Disambiguation (WSD) is enabled, the chosen CUI/Preferred Text reflect that choice.
- Subject and History (patient/family history), Conditional, and Polarity are present
  in XMI and may be surfaced in a future “enhanced” table, but are not included by the
  default `SemanticTableFileWriter` columns.
- TUIs and coding scheme (e.g., `FullClinical_AllTUIs`) are present in XMI
  (`refsem:UmlsConcept` attributes), not included in the default table columns.
 - The workbook `MentionsDetails` sheet adds more attributes, including `Confidence`,
   `CodingScheme` (source vocabulary), and `ConceptScore` for the chosen concept.

WSD behavior and XMI:

- The pipeline uses `tools.wsd.SimpleWsdDisambiguatorAnnotator` which selects a
  single best concept for each mention. It is configured to:
  - Mark the best concept with `disambiguated=true` in XMI.
  - Optionally keep all original candidates, with the best moved to index 0.
    (See the piper files for `KeepAllCandidates`, `MoveBestFirst`, `MarkDisambiguated`.)
- The default tables have no explicit “WSD” column; WSD’s effect appears as the chosen
  `CUI`/`Preferred Text`. Inspect XMI to see the `disambiguated` flag and any retained
  candidates.

Sections:

- “Section” shows `SIMPLE_SEGMENT` if no sectionizer is used. Sectioned pipelines
  (see compare pipelines) include a sectionizer so you’ll see real section names.

Source vocab and codes:

- The bundled `FullClinical_AllTUIs` dictionary populates UMLS CUIs (and TUIs) not
  per–source codes. To get `code_system`/`code` (e.g., SNOMEDCT, RXNORM), switch to a
  dictionary or concept factory that emits per-source codes (e.g., YTEX-backed UMLS),
  then add columns to an enhanced writer.
