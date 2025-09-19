#!/usr/bin/env python3
"""Semantic manifest builder/comparer for cTAKES validation outputs."""

from __future__ import annotations

import argparse
import csv
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional, Tuple
from xml.etree import ElementTree as ET

SCHEMA_VERSION = 2
TEXTSEM_NS = "http:///org/apache/ctakes/typesystem/type/textsem.ecore"
XMI_ID_ATTR = "{http://www.omg.org/XMI}id"

INT_FIELDS = {"begin", "end", "polarity", "uncertainty", "typeID", "historyOf"}
FLOAT_FIELDS = {"confidence", "score"}
BOOL_FIELDS = {"conditional", "generic"}
CONCEPT_BOOL_FIELDS = {"disambiguated"}
CONCEPT_FIELDS = ("codingScheme", "cui", "tui", "preferredText", "score", "disambiguated")
SECTION_KEYS = ("concepts", "cui_counts", "rxnorm", "xmi")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Compare semantic outputs against a baseline manifest.")
    parser.add_argument("--outputs", required=True, help="Path to the pipeline output directory")
    parser.add_argument("--manifest", required=True, help="Path to the semantic manifest JSON")
    parser.add_argument("--report", help="Optional validation report file to append")
    parser.add_argument("--processed-count", type=int, default=0,
                        help="Number of concept files processed (for logging only)")
    return parser.parse_args()


def local_name(tag: str) -> str:
    if tag.startswith("{"):
        return tag.split("}", 1)[1]
    return tag


def convert_value(name: str, value: Optional[str]) -> Any:
    if value is None:
        return None
    value = value.strip()
    if value == "":
        return None
    if name in INT_FIELDS:
        try:
            return int(value)
        except ValueError:
            try:
                return int(float(value))
            except ValueError:
                return value
    if name in FLOAT_FIELDS:
        try:
            return round(float(value), 6)
        except ValueError:
            return value
    if name in BOOL_FIELDS or name in CONCEPT_BOOL_FIELDS:
        lowered = value.lower()
        if lowered in {"true", "1", "yes"}:
            return True
        if lowered in {"false", "0", "no"}:
            return False
        return value
    return value


def safe_int(value: Any) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        try:
            return int(float(value))
        except (TypeError, ValueError):
            return -1


def read_csv(path: Path) -> Tuple[List[str], List[List[str]]]:
    with path.open("r", newline="", encoding="utf-8") as handle:
        reader = csv.reader(handle)
        try:
            header = next(reader)
        except StopIteration:
            return [], []
        rows = [list(row) for row in reader]
    return header, rows


def read_bsv(path: Path) -> Tuple[List[str], List[List[str]]]:
    with path.open("r", encoding="utf-8") as handle:
        lines = [line.rstrip("\n") for line in handle]
    if not lines:
        return [], []
    header = lines[0].split("|")
    rows = [line.split("|") for line in lines[1:] if line.strip()]
    return header, rows


def build_concept_sort_key(header: List[str]):
    index = {name: idx for idx, name in enumerate(header)}
    begin_idx = index.get("core:Begin")
    end_idx = index.get("core:End")
    doc_idx = index.get("core:Document")
    cui_idx = index.get("core:CUI")
    text_idx = index.get("core:Text")

    def key(row: List[str]):
        doc = row[doc_idx] if doc_idx is not None and doc_idx < len(row) else ""
        begin = safe_int(row[begin_idx]) if begin_idx is not None and begin_idx < len(row) else -1
        end = safe_int(row[end_idx]) if end_idx is not None and end_idx < len(row) else -1
        cui = row[cui_idx] if cui_idx is not None and cui_idx < len(row) else ""
        text = row[text_idx] if text_idx is not None and text_idx < len(row) else ""
        return (doc, begin, end, cui, text, row)

    return key


def build_rxnorm_sort_key(header: List[str]):
    index = {name: idx for idx, name in enumerate(header)}
    begin_idx = index.get("Begin")
    end_idx = index.get("End")
    doc_idx = index.get("Document")
    umls_idx = index.get("CUI")
    rx_idx = index.get("RxCUI")
    text_idx = index.get("Text")

    def key(row: List[str]):
        doc = row[doc_idx] if doc_idx is not None and doc_idx < len(row) else ""
        begin = safe_int(row[begin_idx]) if begin_idx is not None and begin_idx < len(row) else -1
        end = safe_int(row[end_idx]) if end_idx is not None and end_idx < len(row) else -1
        umls = row[umls_idx] if umls_idx is not None and umls_idx < len(row) else ""
        rx = row[rx_idx] if rx_idx is not None and rx_idx < len(row) else ""
        text = row[text_idx] if text_idx is not None and text_idx < len(row) else ""
        return (doc, begin, end, umls, rx, text, row)

    return key


def load_concepts(base_dir: Path) -> Dict[str, Dict[str, Any]]:
    result: Dict[str, Dict[str, Any]] = {}
    target = base_dir / "concepts"
    if not target.is_dir():
        return result
    for path in sorted(p for p in target.rglob("*") if p.is_file() and p.suffix.lower() == ".csv"):
        header, rows = read_csv(path)
        sort_key = build_concept_sort_key(header)
        rows.sort(key=sort_key)
        result[path.relative_to(base_dir).as_posix()] = {"header": header, "rows": rows}
    return result


def load_rxnorm(base_dir: Path) -> Dict[str, Dict[str, Any]]:
    result: Dict[str, Dict[str, Any]] = {}
    target = base_dir / "rxnorm"
    if not target.is_dir():
        return result
    for path in sorted(p for p in target.rglob("*") if p.is_file() and p.suffix.lower() == ".csv"):
        header, rows = read_csv(path)
        sort_key = build_rxnorm_sort_key(header)
        rows.sort(key=sort_key)
        result[path.relative_to(base_dir).as_posix()] = {"header": header, "rows": rows}
    return result


def load_cui_counts(base_dir: Path) -> Dict[str, Dict[str, Any]]:
    result: Dict[str, Dict[str, Any]] = {}
    target = base_dir / "cui_counts"
    if not target.is_dir():
        return result
    for path in sorted(p for p in target.rglob("*") if p.is_file() and p.suffix and p.suffix.lower() == ".bsv"):
        header, rows = read_bsv(path)
        rows.sort(key=lambda row: (
            row[0] if row else "",
            safe_int(row[1]) if len(row) > 1 else 0,
            safe_int(row[2]) if len(row) > 2 else 0,
        ))
        result[path.relative_to(base_dir).as_posix()] = {"header": header, "rows": rows}
    return result


def resolve_concepts(ids: Iterable[str], lookup: Dict[str, ET.Element], seen: Optional[set] = None) -> List[Dict[str, Any]]:
    concepts: List[Dict[str, Any]] = []
    if seen is None:
        seen = set()
    for cid in ids:
        if not cid or cid in seen:
            continue
        seen.add(cid)
        element = lookup.get(cid)
        if element is None:
            continue
        lname = local_name(element.tag)
        if lname in {"UmlsConcept", "OntologyConcept"}:
            data: Dict[str, Any] = {"type": lname}
            for field in CONCEPT_FIELDS:
                if field in element.attrib:
                    data[field] = convert_value(field, element.attrib[field])
            concepts.append(data)
        elif lname == "FSArray":
            refs = element.attrib.get("elements", "")
            if refs:
                concepts.extend(resolve_concepts(refs.split(), lookup, seen))
    return concepts


def concept_sort_key(concept: Dict[str, Any]) -> Tuple[Any, ...]:
    score = concept.get("score")
    if isinstance(score, (int, float)):
        score_val: Any = round(float(score), 6)
    elif score is None:
        score_val = 0.0
    else:
        try:
            score_val = round(float(score), 6)
        except (TypeError, ValueError):
            score_val = 0.0
    return (
        concept.get("cui") or "",
        concept.get("tui") or "",
        concept.get("preferredText") or "",
        concept.get("codingScheme") or "",
        score_val,
    )


def mention_sort_key(entry: Dict[str, Any]) -> Tuple[Any, ...]:
    begin = entry.get("begin")
    end = entry.get("end")
    type_name = entry.get("type") or ""
    text = entry.get("text") or ""
    concept_keys = tuple(concept_sort_key(c) for c in entry.get("concepts", []))
    return (
        begin if isinstance(begin, int) else -1,
        end if isinstance(end, int) else -1,
        type_name,
        text,
        concept_keys,
    )


def extract_mention(element: ET.Element, lookup: Dict[str, ET.Element], sofa_text: str) -> Dict[str, Any]:
    data: Dict[str, Any] = {"type": local_name(element.tag)}
    for attr in (
        "begin",
        "end",
        "polarity",
        "uncertainty",
        "conditional",
        "generic",
        "subject",
        "historyOf",
        "confidence",
        "discoveryTechnique",
        "typeID",
        "id",
    ):
        if attr in element.attrib:
            data[attr] = convert_value(attr, element.attrib[attr])
    begin = data.get("begin")
    end = data.get("end")
    if isinstance(begin, int) and isinstance(end, int) and sofa_text:
        if 0 <= begin <= end <= len(sofa_text):
            data["text"] = sofa_text[begin:end]
        else:
            data["text"] = ""
    oc_attr = element.attrib.get("ontologyConceptArr", "")
    concepts: List[Dict[str, Any]] = []
    if oc_attr:
        concepts = resolve_concepts(oc_attr.split(), lookup)
    concepts.sort(key=concept_sort_key)
    data["concepts"] = concepts
    return data


def parse_xmi(path: Path) -> Dict[str, Any]:
    root = ET.parse(path).getroot()
    lookup: Dict[str, ET.Element] = {}
    for child in root:
        identifier = child.attrib.get(XMI_ID_ATTR)
        if identifier:
            lookup[identifier] = child
    sofa_text = ""
    for child in root:
        if child.tag.endswith("Sofa"):
            sofa_text = child.attrib.get("sofaString", "")
            break
    mentions: List[Dict[str, Any]] = []
    for child in root:
        if child.tag.startswith("{"):
            ns, _, _ = child.tag[1:].partition("}")
        else:
            ns = ""
        if ns != TEXTSEM_NS:
            continue
        if not local_name(child.tag).endswith("Mention"):
            continue
        mention = extract_mention(child, lookup, sofa_text)
        mentions.append(mention)
    mentions.sort(key=mention_sort_key)
    return {"mentions": mentions}


def load_xmi(base_dir: Path) -> Dict[str, Dict[str, Any]]:
    result: Dict[str, Dict[str, Any]] = {}
    target = base_dir / "xmi"
    if not target.is_dir():
        return result
    for path in sorted(p for p in target.rglob("*") if p.is_file() and p.suffix.lower() == ".xmi"):
        result[path.relative_to(base_dir).as_posix()] = parse_xmi(path)
    return result


def collect_outputs(base_dir: Path) -> Dict[str, Dict[str, Any]]:
    return {
        "concepts": load_concepts(base_dir),
        "cui_counts": load_cui_counts(base_dir),
        "rxnorm": load_rxnorm(base_dir),
        "xmi": load_xmi(base_dir),
    }


def extract_sections(payload: Dict[str, Any]) -> Dict[str, Dict[str, Any]]:
    return {key: payload.get(key, {}) for key in SECTION_KEYS}


def compute_counts(sections: Dict[str, Dict[str, Any]]) -> Dict[str, int]:
    counts = {key: len(sections.get(key, {})) for key in SECTION_KEYS}
    counts["concept_rows"] = sum(len(entry.get("rows", [])) for entry in sections.get("concepts", {}).values())
    counts["cui_counts_rows"] = sum(len(entry.get("rows", [])) for entry in sections.get("cui_counts", {}).values())
    counts["rxnorm_rows"] = sum(len(entry.get("rows", [])) for entry in sections.get("rxnorm", {}).values())
    counts["xmi_mentions"] = sum(len(entry.get("mentions", [])) for entry in sections.get("xmi", {}).values())
    return counts


def first_difference(current: List[Any], baseline: List[Any]) -> Optional[Tuple[int, Any, Any]]:
    for idx, (cur, base) in enumerate(zip(current, baseline)):
        if cur != base:
            return idx, cur, base
    return None


def describe_row(row: List[str]) -> str:
    preview = " | ".join(row[:5]) if row else ""
    if len(preview) > 96:
        preview = preview[:93] + "..."
    return preview


def describe_mention(entry: Dict[str, Any]) -> str:
    begin = entry.get("begin")
    end = entry.get("end")
    text = (entry.get("text") or "").strip().replace("\n", " ")
    if len(text) > 48:
        text = text[:45] + "..."
    cui = ""
    if entry.get("concepts"):
        cui = entry["concepts"][0].get("cui") or ""
    parts = [entry.get("type") or "?", f"@{begin}-{end}"]
    if cui:
        parts.append(cui)
    if text:
        parts.append(text)
    return " ".join(str(p) for p in parts if p is not None)


def summarise_difference(section: str, key: str, baseline_entry: Dict[str, Any],
                         current_entry: Dict[str, Any]) -> str:
    if section == "xmi":
        base_mentions = baseline_entry.get("mentions", [])
        cur_mentions = current_entry.get("mentions", [])
        if len(base_mentions) != len(cur_mentions):
            return f"mentions {len(cur_mentions)} vs {len(base_mentions)}"
        diff = first_difference(cur_mentions, base_mentions)
        if diff:
            idx, cur, base = diff
            return f"mention[{idx}] {describe_mention(cur)} != {describe_mention(base)}"
    else:
        base_rows = baseline_entry.get("rows", [])
        cur_rows = current_entry.get("rows", [])
        if len(base_rows) != len(cur_rows):
            return f"rows {len(cur_rows)} vs {len(base_rows)}"
        diff = first_difference(cur_rows, base_rows)
        if diff:
            idx, cur, base = diff
            return f"row[{idx}] {describe_row(cur)} != {describe_row(base)}"
    return "content differs"


def compare_data(baseline: Dict[str, Any], current: Dict[str, Any]) -> Tuple[str, Dict[str, Any]]:
    baseline_sections = extract_sections(baseline)
    current_sections = extract_sections(current)
    diffs: Dict[str, Any] = {"missing": {}, "extra": {}, "modified": {}}
    mismatch = False
    for section in SECTION_KEYS:
        base_map = baseline_sections.get(section, {})
        cur_map = current_sections.get(section, {})
        missing = sorted(set(base_map) - set(cur_map))
        extra = sorted(set(cur_map) - set(base_map))
        changed = {}
        for key in sorted(set(base_map) & set(cur_map)):
            if base_map[key] != cur_map[key]:
                changed[key] = summarise_difference(section, key, base_map[key], cur_map[key])
        if missing:
            diffs["missing"][section] = missing
            mismatch = True
        if extra:
            diffs["extra"][section] = extra
            mismatch = True
        if changed:
            diffs["modified"][section] = changed
            mismatch = True
    status = "match" if not mismatch else "diff"
    return status, diffs


def print_diff_summary(diffs: Dict[str, Any]) -> None:
    for section, entries in diffs.get("missing", {}).items():
        if not entries:
            continue
        head = ", ".join(entries[:3])
        more = f" (+{len(entries) - 3} more)" if len(entries) > 3 else ""
        print(f"[validate] Missing {len(entries)} {section} file(s): {head}{more}")
    for section, entries in diffs.get("extra", {}).items():
        if not entries:
            continue
        head = ", ".join(entries[:3])
        more = f" (+{len(entries) - 3} more)" if len(entries) > 3 else ""
        print(f"[validate] New {section} file(s) not in baseline: {head}{more}")
    for section, mapping in diffs.get("modified", {}).items():
        if not mapping:
            continue
        items = list(mapping.items())
        for key, detail in items[:5]:
            print(f"[validate] {section}:{key} -> {detail}")
        if len(items) > 5:
            print(f"[validate] {len(items) - 5} more {section} file(s) differ.")


def write_report_baseline(path: Path, timestamp: str, manifest: Path, counts: Dict[str, int]) -> None:
    with path.open("a", encoding="utf-8") as handle:
        handle.write(f"timestamp={timestamp}\n")
        handle.write("status=baseline-created\n")
        handle.write(f"manifest={manifest}\n")
        handle.write(f"concepts={counts['concepts']}\n")
        handle.write(f"cui_counts={counts['cui_counts']}\n")
        handle.write(f"rxnorm={counts['rxnorm']}\n")
        handle.write(f"xmi={counts['xmi']}\n")
        handle.write("\n")


def write_report_match(path: Path, timestamp: str, manifest: Path, counts: Dict[str, int]) -> None:
    with path.open("a", encoding="utf-8") as handle:
        handle.write(f"timestamp={timestamp}\n")
        handle.write("status=match\n")
        handle.write(f"manifest={manifest}\n")
        handle.write(f"concepts={counts['concepts']}\n")
        handle.write(f"cui_counts={counts['cui_counts']}\n")
        handle.write(f"rxnorm={counts['rxnorm']}\n")
        handle.write(f"xmi={counts['xmi']}\n")
        handle.write("\n")


def write_report_diff(path: Path, timestamp: str, manifest: Path, current_counts: Dict[str, int],
                      baseline_counts: Dict[str, int]) -> None:
    with path.open("a", encoding="utf-8") as handle:
        handle.write(f"timestamp={timestamp}\n")
        handle.write("status=diff\n")
        handle.write(f"manifest={manifest}\n")
        for section in SECTION_KEYS:
            handle.write(f"{section}-current={current_counts[section]}\n")
            handle.write(f"{section}-baseline={baseline_counts[section]}\n")
        handle.write("\n")


def main() -> int:
    args = parse_args()
    outputs_dir = Path(args.outputs).resolve()
    manifest_path = Path(args.manifest).resolve()
    report_path = Path(args.report).resolve() if args.report else None

    current_sections = collect_outputs(outputs_dir)
    current_payload: Dict[str, Any] = {"schema_version": SCHEMA_VERSION, **current_sections}

    timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    current_counts = compute_counts(current_sections)

    baseline_data: Optional[Dict[str, Any]] = None
    legacy_manifest = False
    if manifest_path.exists():
        try:
            baseline_data = json.loads(manifest_path.read_text(encoding="utf-8"))
        except Exception:
            legacy_manifest = True
        else:
            if not isinstance(baseline_data, dict) or baseline_data.get("schema_version") != SCHEMA_VERSION:
                legacy_manifest = True
        if legacy_manifest:
            print(f"[validate] Existing manifest {manifest_path} uses legacy format; capturing new semantic baseline.")
            baseline_data = None

    if baseline_data is None:
        manifest_path.parent.mkdir(parents=True, exist_ok=True)
        manifest_path.write_text(json.dumps(current_payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        print(f"[validate] Baseline manifest saved to {manifest_path}")
        print("[validate] Baseline captured at {} (concepts:{}, cui_counts:{}, rxnorm:{}, xmi:{})".format(
            timestamp,
            current_counts["concepts"],
            current_counts["cui_counts"],
            current_counts["rxnorm"],
            current_counts["xmi"],
        ))
        if report_path:
            write_report_baseline(report_path, timestamp, manifest_path, current_counts)
        return 0

    status, diffs = compare_data(baseline_data, current_payload)
    baseline_counts = compute_counts(extract_sections(baseline_data))

    if status == "match":
        print(f"[validate] Semantic outputs match {manifest_path}")
        if args.processed_count:
            if baseline_counts["concepts"]:
                print(f"[validate] {args.processed_count}/{baseline_counts['concepts']} concept file(s) compared.")
            else:
                print(f"[validate] Processed {args.processed_count} concept file(s); baseline has no concept entries.")
        print("[validate] Files compared -> concepts:{}, cui_counts:{}, rxnorm:{}, xmi:{}".format(
            current_counts["concepts"],
            current_counts["cui_counts"],
            current_counts["rxnorm"],
            current_counts["xmi"],
        ))
        if report_path:
            write_report_match(report_path, timestamp, manifest_path, current_counts)
        return 0

    print(f"[validate] Semantic mismatch detected against {manifest_path}")
    print_diff_summary(diffs)
    print("[validate] Current counts: concepts:{}, cui_counts:{}, rxnorm:{}, xmi:{}".format(
        current_counts["concepts"],
        current_counts["cui_counts"],
        current_counts["rxnorm"],
        current_counts["xmi"],
    ))
    print("[validate] Baseline counts: concepts:{}, cui_counts:{}, rxnorm:{}, xmi:{}".format(
        baseline_counts["concepts"],
        baseline_counts["cui_counts"],
        baseline_counts["rxnorm"],
        baseline_counts["xmi"],
    ))
    if report_path:
        write_report_diff(report_path, timestamp, manifest_path, current_counts, baseline_counts)
    return 1


if __name__ == "__main__":
    sys.exit(main())
