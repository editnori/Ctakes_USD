SHELL := bash

# Stamp for reports directory (default to the existing folder)
STAMP ?= 20250831

.PHONY: help report-analyze report-render report-lint report-all

help:
	 @echo "make report-all        # analyze + render + lint (reports/$(STAMP))"
	 @echo "make report-analyze    # compute Admission summaries (group/TUI/DocTimeRel)"
	 @echo "make report-render     # render slot tables into .rendered.md"
	 @echo "make report-lint       # run acceptance/style checks"
	 @echo "make reports-db        # build consolidated reports_db from mentions.csv (with optional MRSTY)"
	@echo "make clean-rendered    # remove duplicate rendered artifacts"
	@echo "make datasets          # export flat mentions + normalize CUI sets"
	@echo "make audit             # show key report/data locations"

report-analyze:
	 bash scripts/analyze_admission_runs.sh selective_extract/SD5000_Types $(STAMP)

report-render:
	 bash scripts/build_report.sh $(STAMP)

report-lint:
	 bash scripts/lint_report.sh reports/$(STAMP)

report-all: report-analyze report-render report-lint

clean-rendered:
	 bash scripts/cleanup_rendered.sh $(STAMP)

datasets:
	 perl scripts/export_mentions_flat.pl selective_extract/SD5000_Types reports/$(STAMP)/analysis/mentions.csv
	 bash scripts/normalize_cui_sets.sh reports/$(STAMP)/data/cui_sets

reports-db:
	 bash scripts/build_reports_db.sh selective_extract/SD5000_Types $(STAMP) umls_loader/MRSTY.RRF || \
	 bash scripts/build_reports_db.sh selective_extract/SD5000_Types $(STAMP)

audit:
	 @echo "Reports: reports/$(STAMP)/"
	 @echo "Per-type: reports/$(STAMP)/note_types/<Type>/<Type>.rendered.md"
	 @echo "Flat mentions: reports/$(STAMP)/analysis/mentions.csv"
	 @echo "reports_db: reports/$(STAMP)/reports_db (counts_by_*.csv, candidate_ambiguity_by_run.csv, doctimerel_*.csv)"
	 @echo "CUI sets: reports/$(STAMP)/data/cui_sets/*.cuis (+ .norm.csv)"
