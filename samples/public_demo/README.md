Public demo samples

Put 50–100 short, shareable (non‑PHI) clinical‑like `.txt` notes here if you want a built‑in validation pack in the repo. These files are tracked by git (unlike `samples/mimic/*.txt`, which is ignored).

Guidelines:
- Do not include PHI or any restricted dataset content (e.g., MIMIC). Only synthetic or self‑owned text you can redistribute.
- Keep each file small (<10 KB) for fast validation.
- A simple naming convention like `note001.txt` … `note100.txt` is fine.

You can now run `scripts/validate_mimic.sh`; it automatically falls back to `samples/public_demo/` if `samples/mimic/` is empty.

