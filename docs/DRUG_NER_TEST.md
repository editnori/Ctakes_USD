Title: Drug NER Side Test (RxNorm)

Scope
- Run cTAKES Drug NER alone on a folder of notes.
- Write XMI and concept tables; extract RxNorm rows to a single CSV.
- Capture per-document timing for quick cost estimates.

Prereqs
- Set `CTAKES_HOME` to your cTAKES install, e.g. `apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0`.
- Input text files in one folder.

Commands
- Run Drug NER only:
  `bash scripts/run_drug_ner.sh -i <input_dir> -o outputs/drug_ner_test`
- Extract RxNorm rows:
  `bash scripts/extract_rxnorm_from_concepts.sh -p outputs/drug_ner_test`
- Summarize timing:
  `bash scripts/summarize_timing.sh -p outputs/drug_ner_test`

Outputs
- XMI: `outputs/drug_ner_test/xmi/`
- Concepts CSVs: `outputs/drug_ner_test/csv_table_concepts/`
- RxNorm aggregate: `outputs/drug_ner_test/rxnorm/rxnorm.csv`
- Timing TSV: `outputs/drug_ner_test/timing_csv/timing.csv`

Notes
- RxNorm values are pulled from the concepts CSV where `CodingScheme` equals `RXNORM`.
- No PHI is written in examples; review artifacts locally.

