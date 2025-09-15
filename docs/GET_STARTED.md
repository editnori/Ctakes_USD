Title: Get Started Fast

Purpose
- Make first run simple. Minimal steps for: setup, quick MIMIC test, focused main run (Sectioned Core/Relation/Smoking), and the Drug NER side test.

Use With Release + Repo
- Clone the repo:
  `git clone https://github.com/editnori/Ctakes_USD.git CtakesBun && cd CtakesBun`
- Install the cTAKES bundle into this repo (Linux/macOS/Git Bash/WSL):
  `bash scripts/first_time_setup.sh`
- Set `CTAKES_HOME` for this shell:
  `export CTAKES_HOME="$(pwd)/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"`
- Windows: use Git Bash or WSL for all bash scripts.

Quick MIMIC Test (100 notes)
- Put ~100 de-identified notes under `samples/mimic/`.
- Validate only the main pipelines:
  `bash scripts/validate_main.sh`
- Where to look: `outputs/validation_mimic` → per-pipeline run dirs; summary printed at the end.

Focused Main Run (Sectioned Core, Relation, Smoking)
- Status preview:
  `bash scripts/status_main.sh -i <input_dir>`
- Run (autoscale):
  `bash scripts/run_main.sh -i <input_dir> -o outputs/main --reports --autoscale`
- Open results: per-pipeline folders under `outputs/main` with CSVs, tokens, CUI counts, and XMI.

Drug NER Side Test (RxNorm)
- Run Drug NER only:
  `bash scripts/run_drug_ner.sh -i <input_dir> -o outputs/drug_ner_test`
- Extract RxNorm rows:
  `bash scripts/extract_rxnorm_from_concepts.sh -p outputs/drug_ner_test`
- Timing summary:
  `bash scripts/summarize_timing.sh -p outputs/drug_ner_test`

Notes
- Java 11+ required.
- If you already have a cTAKES install, skip `first_time_setup.sh` and set `CTAKES_HOME` to your path.
- No PHI in examples. Use synthetic or redacted notes for testing.

