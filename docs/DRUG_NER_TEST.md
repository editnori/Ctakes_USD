Title: Drug NER Side Test (RxNorm)

Scope
- Run cTAKES Drug NER alone on a folder of notes.
- Write XMI and concept tables; emit per‑doc minimal RxNorm CSVs.
- Capture per-document timing for quick cost estimates.

Prereqs
- Set `CTAKES_HOME` to your cTAKES install, e.g. `apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0`.
- Input text files in one folder.

Commands
- Run Drug NER only:
  `bash scripts/run_drug_ner.sh -i <input_dir> -o outputs/drug_ner_test`
- Aggregate RxNorm rows (optional, merge per‑doc into one file):
  `bash scripts/extract_rxnorm_min.sh -p outputs/drug_ner_test -o outputs/drug_ner_test/rxnorm/rxnorm_min.csv`
- Summarize timing:
  `bash scripts/summarize_timing.sh -p outputs/drug_ner_test`

Outputs
- XMI: `outputs/drug_ner_test/xmi/`
- Concepts CSVs: `outputs/drug_ner_test/csv_table_concepts/`
- RxNorm per‑doc (writer): `outputs/drug_ner_test/rxnorm_min/*.CSV`
- RxNorm aggregate (optional): `outputs/drug_ner_test/rxnorm/rxnorm_min.csv`
- Timing TSV: `outputs/drug_ner_test/timing_csv/timing.csv`

Notes
- The writer `tools.reporting.uima.DrugRxNormCsvWriter` emits only RxNorm‑coded mentions with headers:
  `Document,Begin,End,Text,Section,RxCUI,RxNormName,TUI,SemanticGroup,SemanticTypeLabel`.
- SemanticTypeLabel is derived directly from TUI; SemanticGroup is set to `CHEM` for common drug TUIs.
- The aggregator script is optional; it merges all per‑doc CSVs into a single file for quick review.
- No PHI is written in examples; review artifacts locally.
