#!/usr/bin/env python3
import argparse
import csv
import glob
import os

"""
Extract a minimal RxNorm-only Drug NER CSV from per-document concepts CSVs.

Inputs:
  <run>/csv_table_concepts/*.CSV (from ClinicalConceptCsvWriter)

Output:
  <run>/rxnorm/rxnorm_min.csv with columns:
    Document,Begin,End,Text,Section,RxCUI,RxNormName

Example:
  python3 scripts/extract_rxnorm_min.py -p outputs/drug_ner_test
  python3 scripts/extract_rxnorm_min.py -p outputs/drug_ner_test -o outputs/drug_ner_test/rxnorm/drug_rxnorm_min.csv
"""


def main():
    ap = argparse.ArgumentParser(description="Minimal RxNorm view from csv_table_concepts/*.CSV")
    ap.add_argument('-p', '--path', required=True, help='Run directory (contains csv_table_concepts)')
    ap.add_argument('-o', '--out', default=None, help='Output CSV path (default: <path>/rxnorm/rxnorm_min.csv)')
    args = ap.parse_args()

    in_dir = os.path.join(args.path, 'csv_table_concepts')
    files = sorted(glob.glob(os.path.join(in_dir, '*.CSV'))) + sorted(glob.glob(os.path.join(in_dir, '*.csv')))
    if not files:
        raise SystemExit(f"No concept CSVs found under {in_dir}")

    out_dir = os.path.join(args.path, 'rxnorm') if not args.out else os.path.dirname(args.out)
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    out_path = args.out or os.path.join(args.path, 'rxnorm', 'rxnorm_min.csv')

    fieldnames = ['Document', 'Begin', 'End', 'Text', 'Section', 'RxCUI', 'RxNormName']
    rows = 0
    with open(out_path, 'w', newline='', encoding='utf-8') as fout:
        w = csv.DictWriter(fout, fieldnames=fieldnames)
        w.writeheader()
        for f in files:
            with open(f, 'r', encoding='utf-8', newline='') as fin:
                r = csv.DictReader(fin)
                # Expected upstream fields include: Document, Begin, End, Text, Section, CUI, PreferredText, CodingScheme
                for row in r:
                    scheme = (row.get('CodingScheme') or '').upper()
                    if scheme != 'RXNORM':
                        continue
                    out_row = {
                        'Document': row.get('Document', ''),
                        'Begin': row.get('Begin', ''),
                        'End': row.get('End', ''),
                        'Text': row.get('Text', ''),
                        'Section': row.get('Section', ''),
                        'RxCUI': row.get('CUI', ''),
                        'RxNormName': row.get('PreferredText', '') or row.get('Text', ''),
                    }
                    w.writerow(out_row)
                    rows += 1
    print(f"[rxnorm-min] Wrote {rows} rows -> {out_path}")


if __name__ == '__main__':
    main()

