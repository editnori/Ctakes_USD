Server Run Guide

Overview
- This bundle runs cTAKES 6 pipelines (WSD + Temporal + Coref + Smoking) at scale.
- Use `scripts/run_compare_cluster.sh` to process large corpora in parallel.

Prereqs
- Java 11+ on PATH
- Linux server with fast local disk and /dev/shm available (for HSQL DB)

Quick Start
1) Unpack: tar -xzf ctakes-bundle.tgz && cd CtakesBun
2) Run all pipelines across a root with note-type subfolders (e.g., /data/SD5000_1):
   bash scripts/run_compare_cluster.sh \
     -i /data/SD5000_1 \
     -o /data/ctakes_out \
     --autoscale --reports

   -i: input root (script treats each immediate subdir with .txt files as a group)
   -o: output base dir
  --autoscale: derive parallelism and heap from host cores/memory
--reports: build per‑pipeline workbooks + parent summary (XLSX)

Tuning for big servers (256 CPUs, ~2TB RAM)
- Start with `--autoscale`. If CPU is underutilized and RAM is free, raise `--max-pipelines` to 2 and verify total threads ≤ core count.

Per-note-type runs (keep outputs separated)
for d in EmergencyDepartmentNote InpatientNote OutpatientNote RadiologyReport AdmissionNote DischargeSummary; do
  bash scripts/run_compare_cluster.sh -i "/data/SD5000_1/$d" -o "/data/ctakes_out/$d" -n 128 -m 8192 -t 4 --reports
done

Notes
- No UMLS API key required; the dictionary uses offline HSQL DB copied to /dev/shm per runner.
- After each pipeline/group completes, outputs are consolidated from shard_* into top-level folders (xmi, bsv_table, csv_table, html_table, cui_list, cui_count). Use --no-consolidate to skip, or --keep-shards to preserve shard_* after consolidation.
- Runner writes `timing_csv/timing.csv` and a lightweight `metrics.json` per run for fast reporting.
- For reporting and metrics, the runner saves a parent-level combined log and pipeline file:
  - <run_parent>/run.log: concatenation of all shard_*/run.log
  - <run_parent>/<pipeline>.piper: the tuned piper used by shards
  This ensures the Excel report can compute processing metrics even if shard_* folders are removed.
- All paths are relative to ./apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0 unless CTAKES_HOME is set.


