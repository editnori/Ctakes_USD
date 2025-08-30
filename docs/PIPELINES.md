# Pipelines and Combinations

Use `scripts/run_compare_cluster.sh` to run one or more WSD-enabled pipelines at scale.

- Single pipeline example (WSD fast):
  - `bash scripts/run_compare_cluster.sh -i <input_dir> -o <out_base> --only S_core --reports`
  - Piper: `pipelines/compare/TsSectionedFast_WSD_Compare.piper`
  - Writers: shared block `pipelines/includes/Writers_Xmi_Table.piper` for consistent outputs

- Multiple pipelines (compare families):
  - Sectioned core, relation, temporal, temporal+coref
  - Default (non-sectioned) relation, temporal, temporal+coref, coref
  - Use `--only "S_core S_core_rel S_core_temp S_core_temp_coref D_core_rel D_core_temp D_core_temp_coref D_core_coref"`

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


