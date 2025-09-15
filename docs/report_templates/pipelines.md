Title: Pipelines Summary (Focused)

Scope
- Sectioned Core, Relation, and Smoking only.

Pipelines
- S_core: `pipelines/compare/TsSectionedFast_WSD_Compare.piper`
- S_core_rel: `pipelines/compare/TsSectionedRelation_WSD_Compare.piper`
- S_core_smoke: `pipelines/compare/TsSectionedSmoking_WSD_Compare.piper`

Run Commands
- Validate: `bash scripts/validate_main.sh`
- Full run: `bash scripts/run_main.sh -i <input_dir> -o <output_base> --reports --autoscale`

Notes
- Advanced temporal/coref/default variants are out-of-scope for the main run.

