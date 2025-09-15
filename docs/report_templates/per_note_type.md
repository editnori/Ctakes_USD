Title: Per‑Note‑Type Analysis

Scope
- Compare the three main pipelines across note types.

Inputs
- Output root: `<output_base>` (contains per‑pipeline run folders)
- Concepts CSVs: `<run_dir>/csv_table_concepts/*.csv`
- CUI counts: `<run_dir>/cui_count/*.csv`

Reproduce
- Count docs per note type:
  `fd="<run_dir>"; find "$fd"/xmi -type f -name '*.xmi' | sed -E 's#.*/([^_/]+)_[^/]+$#\1#' | sort | uniq -c`
- Top CUIs by note type:
  `bash scripts/export_top_cuis_by_group.pl -d <run_dir>/cui_count -o <out.csv>`

Findings
- AdmissionNote: <1‑2 sentences>
- DischargeSummary: <1‑2 sentences>
- EmergencyDepartmentNote: <1‑2 sentences>
- InpatientNote: <1‑2 sentences>
- OutpatientNote: <1‑2 sentences>
- RadiologyReport: <1‑2 sentences>

Notes
- Use synthetic identifiers in examples. No PHI.

