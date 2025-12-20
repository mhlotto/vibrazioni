#!/usr/bin/env python3
"""
ietf-draft-difficulty-scan.py

Scan an IETF Internet-Draft (plain text) and flag sections likely to be hard
for LLMs or that should be routed to deterministic parsers/validators.

Outputs JSON to stdout.

Usage:
  python ietf-draft-difficulty-scan.py path/to/draft.txt > report.json

To do:

- Add sections for detection:
  - Appendix / examples sections
  - Labeled figures / listings
  - Algorithm / processing steps
  - Extension / negotiation points
  - Inline registry definitions
  - Error-handling semantics
  - Repeated record definitions
  - Numeric / mathematical constraints
"""

from __future__ import annotations

import json
import re
import sys
from dataclasses import dataclass, asdict
from typing import List, Dict, Any, Optional, Tuple


RFC2119_WORDS = [
    "MUST NOT", "SHALL NOT", "SHOULD NOT",
    "MUST", "SHALL", "SHOULD", "REQUIRED", "RECOMMENDED", "MAY", "OPTIONAL",
]

# Heuristic markers for formats / blocks we generally want to parse deterministically.
GRAMMAR_MARKERS = {
    "abnf": [
        r"\bALPHA\b", r"\bDIGIT\b", r"\bSP\b", r"\bCRLF\b",
        r"^\s*[A-Za-z][A-Za-z0-9\-]*\s*=\s*.+",     # rule = ...
        r"^\s*[A-Za-z][A-Za-z0-9\-]*\s*=\s*/\s*.+", # rule = / ...
    ],
    "cddl": [
        r"^\s*\w[\w\-]*\s*=\s*\{", r"^\s*\w[\w\-]*\s*=\s*\[",
        r"\buint\b", r"\bint\b", r"\btstr\b", r"\bbstr\b", r"\bfloat\b",
        r"=>", r"\b\*\s*\w",  # repetitions / groupings
    ],
    "yang": [
        r"^\s*module\s+\w+\s*\{", r"^\s*container\s+\w+\s*\{",
        r"^\s*leaf\s+\w+\s*\{", r"\bnamespace\b", r"\bprefix\b",
    ],
    "asn1": [
        r"::=", r"\bSEQUENCE\b", r"\bCHOICE\b", r"\bINTEGER\b", r"\bOCTET STRING\b",
    ],
    "json": [
        r'"\s*[^"]+\s*"\s*:',
        r"\btrue\b|\bfalse\b|\bnull\b",
    ],
}

HEX_DUMP_RE = re.compile(r"(?i)\b(?:0x)?[0-9a-f]{2}(?:\s+[0-9a-f]{2}){8,}\b")
URL_RE = re.compile(r"https?://\S+")
REF_RE = re.compile(r"\[(?:RFC|I-D\.)[A-Za-z0-9\.\-]+(?:[:\]]|\])")
SECTION_HDR_RE = re.compile(r"^ ?(\d+(?:\.\d+)*)\.(?:\s{2,}|\s+)(.*\S)\s*$")
PAGE_FOOTER_RE = re.compile(r"^\s*\S+.*\[\s*Page\s+\d+\s*\]\s*$")


@dataclass
class SectionReport:
    number: str
    title: str
    start_line: int
    end_line: int
    line_count: int
    char_count: int
    severity: int
    flags: List[str]
    metrics: Dict[str, Any]
    preview: str


def parse_args(argv: List[str]) -> Dict[str, Any]:
    args = {
        "path": None,
        "mode": "analyze",
        "severity_threshold": 25,
        "replace_with_marker": False,
    }

    it = iter(argv[1:])
    for a in it:
        if a == "--mode":
            args["mode"] = next(it)
        elif a == "--severity-threshold":
            args["severity_threshold"] = int(next(it))
        elif a == "--replace-with-marker":
            args["replace_with_marker"] = True
        else:
            args["path"] = a

    if not args["path"]:
        raise ValueError("Missing input draft path")

    return args


def read_text(path: str) -> str:
    with open(path, "r", encoding="utf-8", errors="replace") as f:
        return f.read()


def strip_page_artifacts(lines: List[str]) -> List[str]:
    # Drop page footers like "Author ... [Page 12]"
    out = []
    for ln in lines:
        if PAGE_FOOTER_RE.match(ln):
            continue
        out.append(ln)
    return out


def find_sections(lines: List[str]) -> List[Tuple[str, str, int]]:
    """
    Return list of (section_number, title, start_line_idx).

    Matches RFC / Internet-Draft headings like:
      1.  Overview
      19.6.  CRYPTO Frames
      A.1.  (Appendices are not handled by this numeric matcher)

    Heuristics:
      - Require heading to start at column 0 (skip indented ToC entries)
      - Require previous line blank-ish AND next line blank-ish (body headings usually surrounded by blank lines)
    """
    sections: List[Tuple[str, str, int]] = []
    n = len(lines)

    for i, ln in enumerate(lines):
        if not ln:
            continue
        if ln.startswith(("  ", "\t")):
            continue  # likely ToC entry (usually indented) or nested list

        m = SECTION_HDR_RE.match(ln.rstrip("\n"))
        if not m:
            continue

        number, title = m.group(1), m.group(2).strip()
        if len(title) < 3:
            continue

        prev_blank = (i == 0) or (lines[i - 1].strip() == "")
        next_blank = (i + 1 < n) and (lines[i + 1].strip() == "")
        if prev_blank and next_blank:
            sections.append((number, title, i))

    return sections



def split_into_sections(lines: List[str]) -> List[Tuple[str, str, int, int, List[str]]]:
    """
    Returns list of (number, title, start_idx, end_idx, section_lines).
    If no section headers are found, treat whole doc as one section.
    """
    hdrs = find_sections(lines)
    if not hdrs:
        return [("0", "Document", 0, len(lines) - 1, lines)]

    out = []
    for idx, (num, title, start) in enumerate(hdrs):
        end = (hdrs[idx + 1][2] - 1) if idx + 1 < len(hdrs) else (len(lines) - 1)
        out.append((num, title, start, end, lines[start:end + 1]))
    return out

def is_bad_section(report: SectionReport, severity_threshold: int) -> bool:
    if report.severity >= severity_threshold:
        return True
    if "route_to_deterministic_parser" in report.flags:
        return True
    if "wire_format_or_syntax_section" in report.flags:
        return True
    if "iana_registry_section" in report.flags:
        return True
    return False


def emit_filtered_draft(
    lines: List[str],
    reports: List[SectionReport],
    severity_threshold: int,
    replace_with_marker: bool,
) -> str:
    """
    Return the draft text with 'bad' sections removed or replaced.
    """
    bad_ranges = []
    for r in reports:
        if is_bad_section(r, severity_threshold):
            # convert back to 0-based
            bad_ranges.append((r.start_line - 1, r.end_line - 1, r))

    bad_ranges.sort()
    out_lines = []
    i = 0

    for start, end, rep in bad_ranges:
        # emit everything before this bad section
        while i < start:
            out_lines.append(lines[i])
            i += 1

        if replace_with_marker:
            marker = (
                f"\n[REMOVED SECTION {rep.number}: {rep.title}]\n"
                f"[severity={rep.severity}, flags={','.join(rep.flags)}]\n\n"
            )
            out_lines.append(marker)

        # skip the bad section
        i = end + 1

    # emit remainder
    while i < len(lines):
        out_lines.append(lines[i])
        i += 1

    return "".join(out_lines)



def count_rfc2119(text: str) -> Dict[str, int]:
    counts = {w: 0 for w in RFC2119_WORDS}
    pattern = re.compile(r"\b(" + "|".join(re.escape(w) for w in RFC2119_WORDS) + r")\b")
    for match in pattern.findall(text):
        counts[match] += 1
    counts["TOTAL"] = sum(counts.values())
    return counts


def detect_tables(lines: List[str]) -> bool:
    # Very common in IETF text: ASCII tables with lots of '|' or repeated '-' lines.
    pipe_lines = sum(1 for ln in lines if ln.count("|") >= 2)
    dash_runs = sum(1 for ln in lines if re.match(r"^\s*[-=]{8,}\s*$", ln))
    return pipe_lines >= 3 or dash_runs >= 3


def detect_ascii_art(lines: List[str]) -> bool:
    # Diagrams / state machines often have many non-alnum chars and long lines.
    weird = 0
    for ln in lines:
        s = ln.rstrip("\n")
        if len(s) >= 80 and re.search(r"[<>\[\]\(\)\-\+\|\\/]{6,}", s):
            weird += 1
    return weird >= 3

FORMAT_DIAGRAM_RE = re.compile(r"^\s*[A-Za-z0-9_ \-]+(?:Packet|Frame|Parameters?)\s*\{\s*$")
FIGURE_RE = re.compile(r"^\s*Figure\s+\d+:", re.IGNORECASE)

def detect_format_diagrams(lines: List[str]) -> bool:
    hit = 0
    for ln in lines:
        if FORMAT_DIAGRAM_RE.match(ln) or FIGURE_RE.match(ln):
            hit += 1
    return hit >= 2  # tune


def detect_grammar_kind(text: str, lines: List[str]) -> List[str]:
    kinds = []
    # Favor fenced-code language hints if present
    if "```abnf" in text.lower():
        kinds.append("abnf")
    if "```cddl" in text.lower():
        kinds.append("cddl")
    if "```yang" in text.lower():
        kinds.append("yang")
    if "```asn1" in text.lower() or "```asn.1" in text.lower():
        kinds.append("asn1")
    if "```json" in text.lower():
        kinds.append("json")

    # Heuristic patterns
    joined = "\n".join(lines)
    for kind, patterns in GRAMMAR_MARKERS.items():
        hits = 0
        for pat in patterns:
            if re.search(pat, joined, flags=re.MULTILINE):
                hits += 1
        # Threshold: at least 2 distinct pattern hits to avoid accidental matches
        if hits >= 2 and kind not in kinds:
            kinds.append(kind)

    return kinds


def compute_severity(metrics: Dict[str, Any], flags: List[str]) -> int:
    """
    Severity 0-100. Tuned to surface sections that benefit from deterministic parsing
    or careful review.
    """
    score = 0
    # Big structural signals
    if metrics.get("has_tables"):
        score += 12
    if metrics.get("has_ascii_art"):
        score += 10
    if metrics.get("has_format_diagrams"):
        score += 14

    # grammar blocks / schemas
    if metrics.get("grammar_kinds"):
        score += 18 + 6 * max(0, len(metrics["grammar_kinds"]) - 1)
    if metrics.get("has_hex_dump"):
        score += 12

    # Density signals
    rfc_total = metrics.get("rfc2119", {}).get("TOTAL", 0)
    if rfc_total >= 10:
        score += 14
    elif rfc_total >= 5:
        score += 9
    elif rfc_total >= 2:
        score += 4

    if metrics.get("crossref_count", 0) >= 12:
        score += 10
    elif metrics.get("crossref_count", 0) >= 6:
        score += 6

    if metrics.get("url_count", 0) >= 8:
        score += 4

    # Long lines / formatting risk
    if metrics.get("long_line_count", 0) >= 10:
        score += 8
    elif metrics.get("long_line_count", 0) >= 4:
        score += 4

    # Size of section (harder for context / attention)
    if metrics.get("line_count", 0) >= 250:
        score += 8
    elif metrics.get("line_count", 0) >= 150:
        score += 5

    # Cap + small bump if many flags
    score += min(8, max(0, len(flags) - 2))
    return max(0, min(100, score))


def build_flags(metrics: Dict[str, Any]) -> List[str]:
    flags: List[str] = []

    if metrics["grammar_kinds"]:
        flags.append(f"contains_grammar_blocks:{','.join(metrics['grammar_kinds'])}")
        flags.append("route_to_deterministic_parser")

    if metrics["has_tables"]:
        flags.append("contains_tables")
    if metrics["has_ascii_art"]:
        flags.append("contains_ascii_diagrams_or_state_machines")
    if metrics.get("has_format_diagrams"):
        flags.append("contains_wire_format_diagrams")
        flags.append("route_to_deterministic_parser")
    if metrics["has_hex_dump"]:
        flags.append("contains_hex_dump_or_binary_example")

    rfc_total = metrics["rfc2119"]["TOTAL"]
    if rfc_total >= 10:
        flags.append("high_normative_density")
    elif rfc_total >= 5:
        flags.append("moderate_normative_density")

    if metrics["crossref_count"] >= 12:
        flags.append("heavy_cross_references")
    elif metrics["crossref_count"] >= 6:
        flags.append("moderate_cross_references")

    if metrics["long_line_count"] >= 10:
        flags.append("many_long_lines_formatting_sensitive")
    elif metrics["long_line_count"] >= 4:
        flags.append("some_long_lines_formatting_sensitive")

    # Common “hard” sections by title
    title_l = metrics["title"].lower()
    if any(k in title_l for k in ["security considerations", "privacy considerations"]):
        flags.append("security_privacy_reasoning_section")
    if any(k in title_l for k in ["iana considerations", "iana"]):
        flags.append("iana_registry_section")
    if any(k in title_l for k in ["formal", "syntax", "encoding", "message format", "wire"]):
        flags.append("wire_format_or_syntax_section")

    return flags


def preview_text(lines: List[str], max_chars: int = 300) -> str:
    txt = "\n".join(ln.rstrip("\n") for ln in lines).strip()
    txt = re.sub(r"\s+\n", "\n", txt)
    if len(txt) <= max_chars:
        return txt
    return txt[: max_chars - 3] + "..."


def analyze_section(number: str, title: str, start: int, end: int, lines: List[str]) -> SectionReport:
    text = "\n".join(lines)
    long_line_count = sum(1 for ln in lines if len(ln.rstrip("\n")) >= 100)
    grammar_kinds = detect_grammar_kind(text, lines)
    has_tables = detect_tables(lines)
    has_ascii_art = detect_ascii_art(lines)
    has_format_diagrams = detect_format_diagrams(lines)
    has_hex_dump = bool(HEX_DUMP_RE.search(text))
    crossref_count = len(REF_RE.findall(text))
    url_count = len(URL_RE.findall(text))
    rfc = count_rfc2119(text)

    metrics: Dict[str, Any] = {
        "title": title,
        "line_count": len(lines),
        "char_count": len(text),
        "long_line_count": long_line_count,
        "grammar_kinds": grammar_kinds,
        "has_tables": has_tables,
        "has_ascii_art": has_ascii_art,
        "has_format_diagrams": has_format_diagrams,
        "has_hex_dump": has_hex_dump,
        "crossref_count": crossref_count,
        "url_count": url_count,
        "rfc2119": rfc,
    }

    flags = build_flags(metrics)
    severity = compute_severity(metrics, flags)

    return SectionReport(
        number=number,
        title=title,
        start_line=start + 1,  # 1-based
        end_line=end + 1,
        line_count=len(lines),
        char_count=len(text),
        severity=severity,
        flags=flags,
        metrics={
            # keep metrics but remove duplicative title
            k: v for k, v in metrics.items() if k != "title"
        },
        preview=preview_text(lines),
    )


def main(argv: List[str]) -> int:
    try:
        args = parse_args(argv)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        print(
            "Usage: python ietf_draft-difficulty-scan.py "
            "[--mode analyze|filter] "
            "[--severity-threshold N] "
            "[--replace-with-marker] "
            "path/to/draft.txt",
            file=sys.stderr,
        )
        return 2

    raw = read_text(args["path"])
    lines = strip_page_artifacts(raw.splitlines(True))

    sections = split_into_sections(lines)
    reports = []
    for num, title, start, end, sec_lines in sections:
        rep = analyze_section(num, title, start, end, sec_lines)
        reports.append(rep)

    if args["mode"] == "filter":
        filtered = emit_filtered_draft(
            lines,
            reports,
            args["severity_threshold"],
            args["replace_with_marker"],
        )
        sys.stdout.write(filtered)
        return 0

    # default: analyze (JSON)
    hotspots = sorted(reports, key=lambda r: r.severity, reverse=True)[:12]
    summary = {
        "input": {"path": args["path"]},
        "section_count": len(reports),
        "hotspots": [
            {
                "number": r.number,
                "title": r.title,
                "severity": r.severity,
                "flags": r.flags,
                "start_line": r.start_line,
                "end_line": r.end_line,
            }
            for r in hotspots
            if r.severity >= 20
        ],
    }

    out = {
        "summary": summary,
        "sections": [asdict(r) for r in reports],
    }

    json.dump(out, sys.stdout, indent=2, ensure_ascii=False)
    sys.stdout.write("\n")
    return 0



if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
