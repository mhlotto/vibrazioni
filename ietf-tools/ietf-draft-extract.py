#!/usr/bin/env python3

# vibe trash
# extract sections of ietf draft txt file format
# see main for usage

import re
import sys
import argparse
import json
from pathlib import Path

# Match headings like:
#   4.1.  General Scope
#   2.1.3.4.  Status 400 - Invalid Client Request
SECTION_RE = re.compile(r"^(\d+(?:\.\d+)*)(?:\.)?\s+(.*\S)\s*$")

# Typical IETF boilerplate patterns
PAGE_HEADER_RE = re.compile(
    r"^\s*(?:Internet-Draft|IETF|RFC)\b.*\s{2,}\S.*$",
    re.I,
)
PAGE_FOOTER_RE = re.compile(r"^\s*.*\[\s*Page\s+\d+\s*\]\s*$", re.I)
TOC_DOT_LEADER_RE = re.compile(r"\.{2,}\s*\d+\s*$")


def parse_sections(lines):
    sections = []
    n = len(lines)
    for i, line in enumerate(lines):
        m = SECTION_RE.match(line)
        if m:
            if TOC_DOT_LEADER_RE.search(line):
                continue
            prev_blank = (i == 0) or (lines[i - 1].strip() == "")
            next_blank = (i + 1 >= n) or (lines[i + 1].strip() == "")
            if not (prev_blank and next_blank):
                continue
            num = m.group(1)   # no trailing dot
            title = m.group(2)
            sections.append((num, title, i))
    return sections


def normalize_query(q: str) -> str:
    q = q.strip()
    num_match = re.fullmatch(r"\d+(?:\.\d+)*\.?", q)
    if num_match and q.endswith("."):
        return q[:-1]
    return q


def find_section(sections, query):
    q = normalize_query(query)

    # Exact match
    for sec in sections:
        if sec[0] == q:
            return sec

    # Prefix match (4.1 → first 4.1.*)
    for sec in sections:
        if sec[0].startswith(q + "."):
            return sec

    # Title match
    qt = query.lower().strip()
    for sec in sections:
        if qt and qt in sec[1].lower():
            return sec

    return None


def extract(lines, sections, target):
    idx = sections.index(target)
    start = target[2]

    if idx + 1 < len(sections):
        end = sections[idx + 1][2]
    else:
        end = len(lines)

    return lines[start:end]


def clean_lines(lines):
    cleaned = []
    for line in lines:
        if line == "\f":
            continue
        if PAGE_HEADER_RE.match(line):
            continue
        if PAGE_FOOTER_RE.match(line):
            continue
        if re.fullmatch(r"-{5,}", line.strip()):
            continue
        line = re.sub(r"\s+\[Page\s+\d+\]$", "", line)
        cleaned.append(line)
    normalized = []
    prev_blank = False
    for line in cleaned:
        is_blank = line.strip() == ""
        if is_blank and prev_blank:
            continue
        normalized.append(line)
        prev_blank = is_blank
    while normalized and normalized[0].strip() == "":
        normalized.pop(0)
    while normalized and normalized[-1].strip() == "":
        normalized.pop()
    return normalized


def print_toc(sections, json_mode=False):
    if json_mode:
        out = [{"number": n, "title": t} for (n, t, _) in sections]
        print(json.dumps(out, indent=2))
    else:
        width = max(len(n) for (n, _, _) in sections) if sections else 4
        for (num, title, _) in sections:
            print(f"{num.ljust(width)}  {title}")


def main():
    ap = argparse.ArgumentParser(description="Extract sections or TOC from IETF draft .txt files.")
    ap.add_argument("file", help="Path to draft .txt file")
    ap.add_argument(
        "query",
        nargs="?",
        default=None,
        help="Section number/title or comma-separated list, e.g. '4.1,4.2.1'"
    )
    ap.add_argument("--toc", action="store_true", help="Print the Table of Contents and exit")
    ap.add_argument("-j", "--json", action="store_true", help="Output JSON")
    ap.add_argument("--clean", action="store_true", help="Remove IETF draft headers/footers/boilerplate")
    ap.add_argument(
        "--full-clean",
        action="store_true",
        help="Output entire draft with headers/footers/boilerplate removed (no section query)",
    )
    args = ap.parse_args()

    text = Path(args.file).read_text(encoding="utf-8", errors="replace")
    lines = text.splitlines()

    if args.full_clean:
        if args.toc:
            print("Cannot combine --full-clean with --toc.", file=sys.stderr)
            sys.exit(1)
        if args.query is not None:
            print("No section query should be provided with --full-clean.", file=sys.stderr)
            sys.exit(1)
        if args.json:
            print("--json is not supported with --full-clean.", file=sys.stderr)
            sys.exit(1)
        print("\n".join(clean_lines(lines)))
        return

    sections = parse_sections(lines)

    # --- TOC MODE ---
    if args.toc:
        print_toc(sections, json_mode=args.json)
        return

    # If no query and not TOC → error
    if args.query is None:
        print("No section query provided (and --toc not specified).", file=sys.stderr)
        sys.exit(1)

    raw_queries = [q.strip() for q in args.query.split(",") if q.strip()]
    if not raw_queries:
        print("No valid section queries provided.", file=sys.stderr)
        sys.exit(1)

    targets = []
    missing = []
    for q in raw_queries:
        t = find_section(sections, q)
        if not t:
            missing.append(q)
        else:
            targets.append(t)

    if missing:
        print(f"No section(s) matching: {', '.join(repr(m) for m in missing)}", file=sys.stderr)
        sys.exit(1)

    extracted_chunks = []
    json_sections = []

    for t in targets:
        chunk = extract(lines, sections, t)
        if args.clean:
            chunk = clean_lines(chunk)
        text = "\n".join(chunk)
        extracted_chunks.append(text)

        json_sections.append({
            "section_number": t[0],
            "title": t[1],
            "content": text,
        })

    if args.json:
        print(json.dumps(json_sections, indent=2))
    else:
        print("\n\n".join(extracted_chunks))


if __name__ == "__main__":
    main()
