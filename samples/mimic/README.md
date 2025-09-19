Place ~100 de-identified MIMIC notes here as plain `.txt` files for quick validation.

Notes:
- `.txt` files are ignored by git so you can keep local packs without polluting commits.
- `scripts/validate_mimic.sh` skips runs when this directory is empty and otherwise builds/compares the baseline manifest.
