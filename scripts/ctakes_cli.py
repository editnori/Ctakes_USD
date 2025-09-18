#!/usr/bin/env python3
"""Interactive runner for cTAKES pipelines.

Prompts for pipeline, discovers note directories, and optionally mirrors the
input layout into the output tree. Supports background execution via the
existing Bash helpers and records metadata so progress can be queried later.
"""

from __future__ import annotations

import argparse
import datetime as dt
from datetime import timezone
import json
import os
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Iterable, List, Optional

PIPELINES = [
    "core",
    "sectioned",
    "smoke",
    "drug",
    "core_sectioned_smoke",
]
RUN_RECORDS_DIR = Path("outputs/run_records")
DEFAULT_INPUT_ROOT = Path("inputs")
DEFAULT_OUTPUT_ROOT = Path("outputs")

PROMPT_DIVIDER = "=" * 60


@dataclass
class NoteSet:
    relative: Path
    path: Path
    count: int

    @property
    def label(self) -> str:
        if self.relative in (Path("."), Path()):
            return "(root)"
        return str(self.relative).replace(os.sep, "/")


def discover_note_sets(root: Path) -> List[NoteSet]:
    if not root.exists():
        raise FileNotFoundError(f"Input root '{root}' does not exist")
    if not root.is_dir():
        raise NotADirectoryError(f"Input root '{root}' is not a directory")
    counts: Dict[Path, int] = {}
    for file in root.rglob("*.txt"):
        parent = file.parent
        try:
            rel = parent.relative_to(root)
        except ValueError:
            rel = Path(".")
        counts[rel] = counts.get(rel, 0) + 1
    if not counts:
        raise FileNotFoundError(f"No .txt files found under '{root}'")
    note_sets = [NoteSet(relative=rel, path=root if rel in (Path("."), Path()) else root / rel, count=count)
                 for rel, count in counts.items()]
    note_sets.sort(key=lambda ns: (str(ns.relative).lower(), ns.label))
    return note_sets


def prompt_choice(prompt: str, choices: List[str], default: Optional[str] = None) -> str:
    while True:
        print(prompt)
        for idx, choice in enumerate(choices, 1):
            default_marker = " (default)" if default and choice == default else ""
            print(f"  {idx}. {choice}{default_marker}")
        raw = input("Select option: ").strip()
        if not raw and default:
            return default
        if raw.isdigit():
            idx = int(raw)
            if 1 <= idx <= len(choices):
                return choices[idx - 1]
        if raw in choices:
            return raw
        print("Invalid selection. Please try again.\n")


def prompt_yes_no(question: str, default: bool = False) -> bool:
    suffix = "[Y/n]" if default else "[y/N]"
    while True:
        raw = input(f"{question} {suffix}: ").strip().lower()
        if not raw:
            return default
        if raw in {"y", "yes"}:
            return True
        if raw in {"n", "no"}:
            return False
        print("Please answer yes or no.\n")


def ensure_dir(path: Path) -> None:
    path.mkdir(parents=True, exist_ok=True)


def write_record(record: Dict) -> None:
    ensure_dir(RUN_RECORDS_DIR)
    target = RUN_RECORDS_DIR / f"{record['id']}.json"
    target.write_text(json.dumps(record, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def run_subprocess(cmd: List[str], *, env: Optional[Dict[str, str]] = None,
                   background: bool = False, capture: bool = False) -> subprocess.CompletedProcess:
    kwargs: Dict = {"env": env}
    if background or capture:
        kwargs.update(stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    return subprocess.run(cmd, check=False, **kwargs)


def parse_background_result(result: subprocess.CompletedProcess) -> Dict[str, Optional[str]]:
    stdout = (result.stdout or "").strip()
    stderr = (result.stderr or "").strip()
    info = {"pid": None, "log": None, "stdout": stdout, "stderr": stderr}
    if stdout:
        match = re.search(r"output -> (.+?) \(pid (\d+)\)", stdout)
        if match:
            info["log"] = match.group(1).strip()
            info["pid"] = match.group(2)
    return info


def prompt_for_input_root(default: Optional[Path]) -> Path:
    default_display = str(default) if default else None
    while True:
        raw = input(f"Input root directory{f' [{default_display}]' if default_display else ''}: ").strip()
        if not raw and default:
            candidate = default
        else:
            candidate = Path(raw).expanduser()
        if candidate.exists() and candidate.is_dir():
            return candidate
        print(f"Directory '{candidate}' not found. Please try again.\n")


def gather_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Interactive cTAKES pipeline runner")
    parser.add_argument("--pipeline", choices=PIPELINES, help="Pipeline key to execute")
    parser.add_argument("--input", help="Input root directory containing note files")
    parser.add_argument("--output", help="Base output directory")
    parser.add_argument("--match-structure", action="store_true", help="Mirror input subdirectories under the output")
    parser.add_argument("--no-match-structure", action="store_true", help="Disable mirroring (process root as one run)")
    parser.add_argument("--background", action="store_true", help="Run using --background")
    parser.add_argument("--foreground", action="store_true", help="Force foreground execution even if --background is set")
    parser.add_argument("--dry-run", action="store_true", help="Print planned commands without executing")
    parser.add_argument("--yes", action="store_true", help="Accept defaults without interactive prompts when possible")
    return parser.parse_args()


def resolve_match_structure(args: argparse.Namespace) -> bool:
    if args.match_structure and args.no_match_structure:
        raise SystemExit("Cannot specify both --match-structure and --no-match-structure")
    if args.match_structure:
        return True
    if args.no_match_structure:
        return False
    return False  # default unless user opts in interactively


def main() -> int:
    args = gather_args()
    auto_accept = args.yes

    pipeline = args.pipeline
    if not pipeline:
        pipeline = prompt_choice("Select pipeline", PIPELINES, default="sectioned")

    default_input = Path(args.input).expanduser() if args.input else (DEFAULT_INPUT_ROOT if DEFAULT_INPUT_ROOT.exists() else None)
    input_root = Path(args.input).expanduser() if args.input else prompt_for_input_root(default_input)

    try:
        note_sets = discover_note_sets(input_root)
    except (FileNotFoundError, NotADirectoryError) as exc:
        print(f"Error: {exc}")
        return 1

    mirror_structure = resolve_match_structure(args)
    if not mirror_structure:
        # Ask the user if they want to mirror structure when multiple sets exist
        if len(note_sets) > 1 and not args.no_match_structure and not auto_accept:
            mirror_structure = prompt_yes_no(
                "Mirror output directories to match input subdirectories?", default=True
            )

    selected_sets = note_sets
    if mirror_structure and len(note_sets) > 1 and not auto_accept:
        print(PROMPT_DIVIDER)
        print("Discovered note sets:")
        for idx, note_set in enumerate(note_sets, 1):
            print(f"  {idx}. {note_set.label:<30} ({note_set.count} files)")
        selection = input("Enter comma-separated indices to process (blank for all): ").strip()
        if selection:
            chosen: List[NoteSet] = []
            for token in selection.split(','):
                token = token.strip()
                if not token:
                    continue
                if not token.isdigit() or not (1 <= int(token) <= len(note_sets)):
                    print(f"Ignoring invalid selection '{token}'.")
                    continue
                chosen.append(note_sets[int(token) - 1])
            if chosen:
                selected_sets = chosen

    total_docs = sum(ns.count for ns in (selected_sets if mirror_structure else note_sets))
    print(PROMPT_DIVIDER)
    print(f"Pipeline: {pipeline}")
    print(f"Input root: {input_root}")
    print(f"Total notes: {total_docs}")
    if mirror_structure:
        print("Will process note sets individually and mirror their directory names under the output.")
    else:
        if len(note_sets) > 1:
            print("Processing the entire input tree as a single run.")

    default_output = Path(args.output).expanduser() if args.output else (DEFAULT_OUTPUT_ROOT / pipeline)
    if not args.output:
        ensure_dir(default_output)
    else:
        ensure_dir(default_output)
    output_base = default_output

    background = args.background and not args.foreground
    if not args.background and not args.foreground:
        background = False if auto_accept else prompt_yes_no("Run in background using nohup?", default=False)

    dry_run = args.dry_run

    run_id = dt.datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S")
    record = {
        "id": run_id,
        "created": dt.datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "pipeline": pipeline,
        "input_root": str(input_root),
        "output_base": str(output_base),
        "mirror_structure": mirror_structure,
        "background": background,
        "runs": [],
    }

    def build_output_path(note_set: Optional[NoteSet]) -> Path:
        if mirror_structure and note_set is not None and note_set.relative not in (Path("."), Path()):
            return output_base / run_id / note_set.relative
        if mirror_structure:
            return output_base / run_id / "root"
        return output_base / run_id

    commands_to_run: List[Dict] = []

    if mirror_structure:
        total_segments = len(selected_sets)
        for idx, note_set in enumerate(selected_sets, 1):
            out_dir = build_output_path(note_set)
            ensure_dir(out_dir)
            env = os.environ.copy()
            env["RUNNER_INDEX"] = str(idx)
            env["RUNNER_COUNT"] = str(total_segments)
            cmd = ["bash", "scripts/run_pipeline.sh", "--pipeline", pipeline,
                   "-i", str(note_set.path), "-o", str(out_dir)]
            commands_to_run.append({
                "note_set": note_set,
                "cmd": cmd,
                "env": env,
                "output": out_dir,
            })
    else:
        out_dir = build_output_path(None)
        ensure_dir(out_dir)
        env = os.environ.copy()
        env["RUNNER_INDEX"] = "1"
        env["RUNNER_COUNT"] = "1"
        cmd = ["bash", "scripts/run_pipeline.sh", "--pipeline", pipeline,
               "-i", str(input_root), "-o", str(out_dir)]
        commands_to_run.append({
            "note_set": None,
            "cmd": cmd,
            "env": env,
            "output": out_dir,
        })

    if dry_run:
        print(PROMPT_DIVIDER)
        print("Dry run; commands not executed:")
        for entry in commands_to_run:
            print(" ".join(entry["cmd"] + (["--background"] if background else [])))
        return 0

    print(PROMPT_DIVIDER)
    print(f"Launching {len(commands_to_run)} run(s)...")

    for idx, entry in enumerate(commands_to_run, 1):
        note_set: Optional[NoteSet] = entry["note_set"]
        cmd: List[str] = entry["cmd"][:]
        env = entry["env"]
        out_dir: Path = entry["output"]
        meta = {
            "input": str(note_set.path if note_set else input_root),
            "label": note_set.label if note_set else "(root)",
            "output": str(out_dir),
            "command": cmd + (["--background"] if background else []),
            "status": "running" if background else "completed",
            "note_count": note_set.count if note_set else total_docs,
            "start_time": dt.datetime.now(timezone.utc).isoformat(timespec="seconds"),
            "pid": None,
            "log": None,
            "exit_code": None,
            "elapsed_seconds": None,
        }

        full_cmd = cmd + (["--background"] if background else [])
        print(f"[{idx}/{len(commands_to_run)}] {' '.join(full_cmd)}")
        result = run_subprocess(full_cmd, env=env, background=background)

        if background:
            info = parse_background_result(result)
            if result.returncode != 0:
                print(result.stdout)
                print(result.stderr, file=sys.stderr)
                print("Background launch failed.")
                meta["status"] = "failed"
                meta["exit_code"] = result.returncode
            else:
                meta["pid"] = info.get("pid")
                meta["log"] = info.get("log")
                if info.get("stdout"):
                    print(info["stdout"])
                if info.get("stderr"):
                    print(info["stderr"], file=sys.stderr)
        else:
            meta["exit_code"] = result.returncode
            end_time = dt.datetime.now(timezone.utc)
            if result.returncode != 0:
                meta["status"] = "failed"
                print("Run failed (see console output above).", file=sys.stderr)
            else:
                meta["status"] = "completed"
            start_dt = dt.datetime.fromisoformat(meta["start_time"])
            meta["elapsed_seconds"] = int((end_time - start_dt).total_seconds())
            meta["end_time"] = end_time.isoformat(timespec="seconds")

        record["runs"].append(meta)
        write_record(record)

    if background:
        print("\nUse 'python scripts/run_status.py --list' to view running jobs and "
              f"'python scripts/run_status.py --run {run_id}' for details.")
    else:
        print("All runs finished. Metadata recorded under outputs/run_records/.")

    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("\nInterrupted.")
        sys.exit(130)
