Title: Timing Summary

Scope
- Report average processing time per pipeline and P50/P90 document latencies.

Inputs
- Per-pipeline CSV: `<run_dir>/timing_csv/pipeline_timing.csv`
- Per-doc CSV: `<run_dir>/timing_csv/timing.csv`

Quick Commands
- Avg per pipeline: `bash scripts/summarize_timing.sh -p <run_dir>`
- P50/P90 per doc: `bash scripts/summarize_doc_percentiles.sh -p <run_dir>`

Snapshot
- S_core: avg=<ms>, P50=<ms>, P90=<ms>
- S_core_rel: avg=<ms>, P50=<ms>, P90=<ms>
- S_core_smoke: avg=<ms>, P50=<ms>, P90=<ms>

Observations
- Short sentence on variance or outliers (no PHI).

