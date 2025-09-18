#!/usr/bin/env python3
"""Inspect and report on recorded cTAKES runs."""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional

RUN_RECORDS_DIR = Path("outputs/run_records")
DATE_FORMAT = "%Y-%m-%d %H:%M:%S"

PROGRESS_PATTERN = re.compile(r"\[async\] progress (\d+)/(\d+) \((\d+)%\).*?docs (\d+)/(\d+) .*?elapsed (\d+)s.*?(shard-\d+).*")
DONE_PATTERN = re.compile(r"\[pipeline]\[runner=(\d+)/(\d+)] done in (\d+)s")
START_PATTERN = re.compile(r"\[pipeline]\[runner=(\d+)/(\d+)] start (.+)")
FAIL_PATTERN = re.compile(r"\[pipeline]\[runner=(\d+)/(\d+)] failed after (\d+)s.*exit=(\d+)")
ASYNC_DONE_PATTERN = re.compile(r"\[async] all shards completed in (\d+)s")
ASYNC_FAIL_PATTERN = re.compile(r"\[async] completed with (\d+) failure\(s\) in (\d+)s")


def load_record(path: Path) -> Dict:
    return json.loads(path.read_text(encoding="utf-8"))


def list_records() -> List[Path]:
    if not RUN_RECORDS_DIR.exists():
        return []
    return sorted(RUN_RECORDS_DIR.glob("*.json"), key=lambda p: p.stem)


def summarize_records(records: List[Path]) -> None:
    if not records:
        print("No recorded runs found (outputs/run_records).")
        return
    print(f"Found {len(records)} run record(s):")
    for record_path in records:
        record = load_record(record_path)
        runs = record.get("runs", [])
        statuses = {entry.get("status", "unknown") for entry in runs}
        status = ",".join(sorted(statuses)) if statuses else "unknown"
        created = record.get("created", "?")
        print(f"  {record['id']}  pipeline={record.get('pipeline')}  status={status}  created={created}")


def tail_lines(path: Path, limit: int = 200) -> List[str]:
    if not path.exists():
        return []
    with path.open("r", encoding="utf-8", errors="replace") as handle:
        try:
            handle.seek(0, os.SEEK_END)
            size = handle.tell()
            block = min(size, 8192)
            handle.seek(size - block)
            data = handle.read()
        except OSError:
            handle.seek(0)
            data = handle.read()
    lines = data.splitlines()
    return lines[-limit:]


def parse_pipeline_log(log_path: Path) -> Dict[str, Optional[str]]:
    result = {"status": "running", "start": None, "end": None, "elapsed": None}
    lines = tail_lines(log_path)
    for line in lines:
        match = START_PATTERN.search(line)
        if match:
            result["start"] = match.group(3).strip()
        match = DONE_PATTERN.search(line)
        if match:
            result["status"] = "completed"
            result["elapsed"] = match.group(3)
    for line in lines:
        match = FAIL_PATTERN.search(line)
        if match:
            result["status"] = f"failed (exit={match.group(4)})"
            result["elapsed"] = match.group(3)
    if result["start"] and not result["elapsed"]:
        try:
            start_dt = datetime.strptime(result["start"], DATE_FORMAT)
            elapsed = int((datetime.now() - start_dt).total_seconds())
            result["elapsed"] = str(elapsed)
        except ValueError:
            pass
    return result


def parse_async_log(log_path: Path) -> Dict[str, Optional[str]]:
    result = {"status": "running", "progress": None, "elapsed": None}
    lines = tail_lines(log_path)
    for line in reversed(lines):
        match = PROGRESS_PATTERN.search(line)
        if match:
            completed, total, percent, docs_done, docs_total, elapsed, shard = match.groups()
            result["progress"] = (
                f"{completed}/{total} shards ({percent}%), docs {docs_done}/{docs_total}, elapsed {elapsed}s, last={shard}"
            )
            break
    for line in lines:
        match = ASYNC_DONE_PATTERN.search(line)
        if match:
            result["status"] = "completed"
            result["elapsed"] = match.group(1)
    for line in lines:
        match = ASYNC_FAIL_PATTERN.search(line)
        if match:
            result["status"] = f"failed ({match.group(1)} shard errors)"
            result["elapsed"] = match.group(2)
    return result


def describe_run(run: Dict) -> None:
    print(f"- Label: {run.get('label')} (notes: {run.get('note_count', '?')})")
    print(f"  Input: {run.get('input')}")
    print(f"  Output: {run.get('output')}")
    log_path = run.get("log")
    if log_path:
        print(f"  Log: {log_path}")
    else:
        print("  Log: (not available)")
    status = run.get("status", "unknown")
    print(f"  Status: {status}")
    if run.get("pid"):
        print(f"  PID: {run['pid']}")
    if run.get("elapsed_seconds"):
        print(f"  Elapsed: {run['elapsed_seconds']}s")

    if log_path:
        path = Path(log_path)
        if "run_async.log" in path.name:
            summary = parse_async_log(path)
            progress = summary.get("progress")
            if progress:
                print(f"  Progress: {progress}")
            if summary.get("elapsed"):
                print(f"  Total elapsed: {summary['elapsed']}s")
            print(f"  Derived status: {summary.get('status')}")
        elif "run_pipeline.log" in path.name:
            summary = parse_pipeline_log(path)
            if summary.get("start"):
                print(f"  Started: {summary['start']}")
            if summary.get("elapsed"):
                print(f"  Elapsed: {summary['elapsed']}s")
            print(f"  Derived status: {summary.get('status')}")
    print()


def update_run_status(record_path: Path) -> Dict:
    record = load_record(record_path)
    updated = False
    for run in record.get("runs", []):
        log_path = run.get("log")
        if not log_path:
            continue
        path = Path(log_path)
        if not path.exists():
            continue
        if "run_async.log" in path.name:
            summary = parse_async_log(path)
            if summary.get("status") == "completed" and run.get("status") != "completed":
                run["status"] = "completed"
                run["elapsed_seconds"] = summary.get("elapsed")
                updated = True
        elif "run_pipeline.log" in path.name:
            summary = parse_pipeline_log(path)
            if summary.get("status") == "completed" and run.get("status") != "completed":
                run["status"] = "completed"
                run["elapsed_seconds"] = summary.get("elapsed")
                updated = True
            elif summary.get("status", "").startswith("failed") and run.get("status") != "failed":
                run["status"] = summary.get("status")
                run["elapsed_seconds"] = summary.get("elapsed")
                updated = True
    if updated:
        record_path.write_text(json.dumps(record, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return record


def show_run(record_path: Path) -> None:
    record = update_run_status(record_path)
    print(PROMPT_DIVIDER)
    print(f"Run ID: {record['id']}")
    print(f"Pipeline: {record.get('pipeline')}")
    print(f"Created: {record.get('created')}")
    print(f"Background: {record.get('background')}")
    print(f"Input root: {record.get('input_root')}")
    print(f"Output base: {record.get('output_base')}")
    print(PROMPT_DIVIDER)
    for run in record.get("runs", []):
        describe_run(run)


PROMPT_DIVIDER = "=" * 60


def main() -> int:
    parser = argparse.ArgumentParser(description="Check progress of cTAKES runs launched via ctakes_cli.py")
    parser.add_argument("--list", action="store_true", help="List recorded runs")
    parser.add_argument("--run", metavar="RUN_ID", help="Show details for a specific run")
    args = parser.parse_args()

    records = list_records()
    if args.list or not args.run:
        summarize_records(records)
    if args.run:
        record_path = RUN_RECORDS_DIR / f"{args.run}.json"
        if not record_path.exists():
            print(f"Run record '{args.run}' not found in outputs/run_records.")
            return 1
        show_run(record_path)
    return 0


if __name__ == "__main__":
    sys.exit(main())
