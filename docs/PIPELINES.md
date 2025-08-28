# Pipelines and Combinations

Two entry scripts exercise WSD-enabled pipelines:

- `scripts/run_wsd_smoke.sh`: quick core+WSD run on a sample directory.
  - Piper: `pipelines/wsd/TsDefaultFastPipeline_WSD.piper`
  - Writers: shared block from `pipelines/includes/Writers_Xmi_Table.piper`

- `scripts/run_compare_smoke.sh`: runs multiple combinations to compare modules.
  - Pipelines:
    - Sectioned core, relation, temporal, temporal+coref
    - Default (non-sectioned) relation, temporal, temporal+coref, coref
  - Each loads the same unified writers include, so outputs are consistent.

WSD configuration (uniform across pipelines):

- `tools.wsd.SimpleWsdDisambiguatorAnnotator` with:
  - `KeepAllCandidates=true`: retain original candidates in XMI
  - `MoveBestFirst=true`: best concept is at index 0
  - `MarkDisambiguated=true`: best concept has `disambiguated=true`

Outputs per pipeline:

- All pipelines write the same directories documented in `docs/OUTPUTS.md`.
- Relation/Temporal/Coref pipelines add their annotations into the XMI; tables
  still summarize mention-level semantics for quick validation.

Extending coverage:

- To include assertion details (subject/history/conditional) or TUIs in tables,
  add an enhanced table writer across all pipelines for consistent headers.
- To include per-source vocab/code, use a concept source that emits those fields
  and add columns to the enhanced writer.

