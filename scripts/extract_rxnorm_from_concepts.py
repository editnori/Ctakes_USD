#!/usr/bin/env python3
import argparse
import csv
import glob
import os

def main():
    ap = argparse.ArgumentParser(description="Aggregate RXNORM rows from csv_table_concepts/*.CSV into one CSV")
    ap.add_argument('-p','--path', required=True, help='Run directory (contains csv_table_concepts)')
    ap.add_argument('-o','--out', default=None, help='Output CSV path (default: <path>/rxnorm/rxnorm.csv)')
    args = ap.parse_args()

    in_dir = os.path.join(args.path, 'csv_table_concepts')
    files = sorted(glob.glob(os.path.join(in_dir, '*.CSV'))) + sorted(glob.glob(os.path.join(in_dir, '*.csv')))
    if not files:
        raise SystemExit(f"No concept CSVs found under {in_dir}")

    out_dir = os.path.join(args.path, 'rxnorm') if not args.out else os.path.dirname(args.out)
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    out_path = args.out or os.path.join(args.path, 'rxnorm', 'rxnorm.csv')

    wrote_header = False
    count = 0
    with open(out_path, 'w', newline='', encoding='utf-8') as fout:
        w = None
        for f in files:
            with open(f, 'r', encoding='utf-8', newline='') as fin:
                r = csv.DictReader(fin)
                needed = ['Document','Begin','End','Text','CUI','PreferredText','CodingScheme']
                if not wrote_header:
                    w = csv.DictWriter(fout, fieldnames=needed)
                    w.writeheader()
                    wrote_header = True
                for row in r:
                    if (row.get('CodingScheme') or '').upper() == 'RXNORM':
                        w.writerow({k: row.get(k,'') for k in needed})
                        count += 1
    print(f"[rxnorm] Wrote {count} rows -> {out_path}")

if __name__ == '__main__':
    main()

