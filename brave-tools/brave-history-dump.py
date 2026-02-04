#!/usr/bin/env python3
"""
brave_history_dump.py

Dump Brave (Chromium) history from the History SQLite DB.

Targets:
  - visits: raw rows from `visits` (visit events)
  - joined: visits joined to urls (visit events + url/title/etc.)
  - urls:   rows from `urls` (unique URL entries)

Output formats:
  - JSONL (default): one JSON object per line
  - CSV: --csv

Examples:
  python3 brave_history_dump.py urls --limit 500 > urls.jsonl
  python3 brave_history_dump.py urls --since-days 14 --url-like "%github.com%" --csv > urls.csv
  python3 brave_history_dump.py visits --since-days 7 > visits.jsonl
  python3 brave_history_dump.py joined --since-time "2026-02-01T00:00:00Z" --csv > joined.csv

Notes:
  - Chromium timestamps are microseconds since 1601-01-01 UTC.
  - Brave profiles are under .../Default or .../Profile <n>.
"""

import argparse
import csv
import json
import os
import shutil
import sqlite3
import sys
import tempfile
import subprocess
from urllib.parse import urlparse
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional, Tuple


CHROMIUM_EPOCH = datetime(1601, 1, 1, tzinfo=timezone.utc)


def chrome_time_to_datetime(microseconds_since_1601: int) -> Optional[datetime]:
    if microseconds_since_1601 is None:
        return None
    try:
        if int(microseconds_since_1601) <= 0:
            return None
        return CHROMIUM_EPOCH + timedelta(microseconds=int(microseconds_since_1601))
    except Exception:
        return None


def datetime_to_chrome_time(dt: datetime) -> int:
    if dt.tzinfo is None:
        raise ValueError("datetime must be timezone-aware")
    delta = dt.astimezone(timezone.utc) - CHROMIUM_EPOCH
    return int(delta.total_seconds() * 1_000_000)


def brave_user_data_dir() -> Path:
    home = Path.home()

    if os.name == "nt":
        localapp = os.environ.get("LOCALAPPDATA")
        if not localapp:
            raise RuntimeError("LOCALAPPDATA not set")
        return Path(localapp) / "BraveSoftware" / "Brave-Browser" / "User Data"

    if sys.platform == "darwin":
        return home / "Library" / "Application Support" / "BraveSoftware" / "Brave-Browser"

    return home / ".config" / "BraveSoftware" / "Brave-Browser"


def history_db_path(profile: Optional[str] = None) -> Path:
    base = brave_user_data_dir()

    if profile:
        p = base / profile / "History"
        if not p.exists():
            raise FileNotFoundError(f"History DB not found at: {p}")
        return p

    candidates = [base / "Default" / "History"]
    candidates += list(base.glob("Profile */History"))
    for p in candidates:
        if p.exists():
            return p

    raise FileNotFoundError(f"Could not find Brave History DB under: {base}")


def copy_db_for_reading(db_path: Path) -> Path:
    td = tempfile.mkdtemp(prefix="brave-history-")
    dst = Path(td) / "History"
    shutil.copy2(db_path, dst)
    return dst


# ----------------------------
# SQL templates
# ----------------------------

VISITS_SQL = """
SELECT
  id,
  url,
  visit_time,
  from_visit,
  transition,
  segment_id,
  visit_duration
FROM visits
{where}
ORDER BY visit_time DESC
{limit}
"""

JOINED_SQL = """
SELECT
  v.id AS visit_id,
  v.visit_time,
  v.from_visit,
  v.transition,
  v.visit_duration,
  v.segment_id,

  u.id AS url_id,
  u.url,
  u.title,
  u.visit_count,
  u.typed_count,
  u.last_visit_time,
  u.hidden
FROM visits v
JOIN urls u ON u.id = v.url
{where}
ORDER BY v.visit_time DESC
{limit}
"""

URLS_SQL = """
SELECT
  id,
  url,
  title,
  visit_count,
  typed_count,
  last_visit_time,
  hidden
FROM urls
{where}
ORDER BY last_visit_time DESC
{limit}
"""


# ----------------------------
# WHERE builder
# ----------------------------

def parse_iso8601(s: str) -> datetime:
    s2 = s.strip().replace("Z", "+00:00")
    dt = datetime.fromisoformat(s2)
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt


def build_where_and_params(
    since_days: Optional[int],
    since_time: Optional[str],
    until_time: Optional[str],
    url_like: Optional[str],
    time_expr: str,
    url_expr: Optional[str],
) -> Tuple[str, List[Any]]:
    clauses = []
    params: List[Any] = []

    if since_days is not None:
        dt = datetime.now(timezone.utc) - timedelta(days=since_days)
        clauses.append(f"{time_expr} >= ?")
        params.append(datetime_to_chrome_time(dt))

    if since_time:
        dt = parse_iso8601(since_time)
        clauses.append(f"{time_expr} >= ?")
        params.append(datetime_to_chrome_time(dt))

    if until_time:
        dt = parse_iso8601(until_time)
        clauses.append(f"{time_expr} <= ?")
        params.append(datetime_to_chrome_time(dt))

    if url_like:
        if not url_expr:
            raise ValueError("--url-like is not supported for this target.")
        clauses.append(f"{url_expr} LIKE ?")
        params.append(url_like)

    where = ""
    if clauses:
        where = "WHERE " + " AND ".join(clauses)
    return where, params


# ----------------------------
# Output helpers
# ----------------------------

def rows_to_dicts(cursor: sqlite3.Cursor, rows: List[Tuple[Any, ...]]) -> List[Dict[str, Any]]:
    colnames = [d[0] for d in cursor.description]
    return [{colnames[i]: r[i] for i in range(len(colnames))} for r in rows]


def normalize_time_fields(d: Dict[str, Any], time_keys: Iterable[str]) -> Dict[str, Any]:
    for k in time_keys:
        if k in d and d[k] is not None:
            dt = chrome_time_to_datetime(int(d[k]))
            d[f"{k}_utc_iso"] = dt.isoformat() if dt else None
    return d


def write_jsonl(records: List[Dict[str, Any]], fp) -> None:
    for r in records:
        fp.write(json.dumps(r, ensure_ascii=False) + "\n")


def write_csv(records: List[Dict[str, Any]], fp) -> None:
    if not records:
        return
    fieldnames = list(records[0].keys())
    w = csv.DictWriter(fp, fieldnames=fieldnames)
    w.writeheader()
    for r in records:
        w.writerow(r)


# ----------------------------
# Graph helpers
# ----------------------------

def dot_escape(s: str) -> str:
    return s.replace("\\", "\\\\").replace("\"", "\\\"").replace("\n", "\\n")


def record_label(rec: Dict[str, Any]) -> str:
    visit_id = rec.get("visit_id")
    url = rec.get("url") or ""
    host = urlparse(url).netloc or url
    title = rec.get("title") or ""
    time_iso = rec.get("visit_time_utc_iso") or ""
    title_short = title[:40] + ("..." if len(title) > 40 else "")
    host_short = host[:40] + ("..." if len(host) > 40 else "")
    return f"{visit_id}\n{host_short}\n{title_short}\n{time_iso}"


def build_dot(records: List[Dict[str, Any]]) -> str:
    nodes: Dict[int, str] = {}
    edges: List[Tuple[int, int]] = []
    missing: Dict[int, str] = {}

    for rec in records:
        vid = rec.get("visit_id")
        if vid is None:
            continue
        try:
            vid_int = int(vid)
        except Exception:
            continue
        nodes[vid_int] = record_label(rec)

    for rec in records:
        vid = rec.get("visit_id")
        from_visit = rec.get("from_visit") or 0
        if vid is None:
            continue
        try:
            vid_int = int(vid)
            from_int = int(from_visit)
        except Exception:
            continue
        if from_int <= 0:
            continue
        edges.append((from_int, vid_int))
        if from_int not in nodes:
            missing[from_int] = f"{from_int}\n(ref)"

    lines: List[str] = [
        "digraph G {",
        "  rankdir=LR;",
        "  node [shape=box, fontname=\"Helvetica\", fontsize=10, color=\"#444444\"];",
        "  edge [color=\"#888888\"];",
    ]

    for vid, label in nodes.items():
        lines.append(f"  \"{vid}\" [label=\"{dot_escape(label)}\"];")
    for vid, label in missing.items():
        lines.append(f"  \"{vid}\" [label=\"{dot_escape(label)}\", shape=ellipse, style=dashed, color=\"#777777\"];")
    for src, dst in edges:
        lines.append(f"  \"{src}\" -> \"{dst}\";")

    lines.append("}")
    return "\n".join(lines)


def write_graph_html(records: List[Dict[str, Any]], out_path: str) -> None:
    data_json = json.dumps(records).replace("</", "<\\/")
    template = """<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Brave History Graph</title>
  <style>
    :root {
      color-scheme: dark;
    }
    body {
      margin: 0;
      font-family: -apple-system, BlinkMacSystemFont, "Helvetica Neue", Helvetica, Arial, sans-serif;
      background: #0c0f14;
      color: #e6e6e6;
    }
    #wrap {
      display: grid;
      grid-template-columns: 1fr 320px;
      height: 100vh;
    }
    #main {
      position: relative;
      overflow: auto;
      border-right: 1px solid #1f2630;
    }
    #controls {
      display: flex;
      gap: 12px;
      align-items: center;
      padding: 10px 14px;
      background: #121823;
      position: sticky;
      top: 0;
      z-index: 2;
      border-bottom: 1px solid #1f2630;
    }
    #details {
      padding: 16px;
      overflow: auto;
    }
    #details h2 {
      margin-top: 0;
      font-size: 16px;
    }
    .muted {
      color: #9aa4b2;
    }
    .node {
      cursor: pointer;
      stroke-width: 1.5;
    }
    .edge {
      stroke: #566274;
      stroke-width: 1;
      opacity: 0.7;
    }
    .lane-label {
      fill: #9aa4b2;
      font-size: 11px;
    }
  </style>
</head>
<body>
  <div id="wrap">
    <div id="main">
      <div id="controls">
        <div>Scale</div>
        <input id="scale" type="range" min="1" max="20" value="6">
        <div class="muted" id="summary"></div>
      </div>
      <svg id="graph" xmlns="http://www.w3.org/2000/svg"></svg>
    </div>
    <div id="details">
      <h2>Visit details</h2>
      <div class="muted">Click a node</div>
      <pre id="detail-json"></pre>
    </div>
  </div>
<script>
const RECORDS = __DATA_JSON__;

const svg = document.getElementById("graph");
const scaleInput = document.getElementById("scale");
const summaryEl = document.getElementById("summary");
const detailJson = document.getElementById("detail-json");

function byIdMap(records) {
  const map = new Map();
  for (const r of records) {
    map.set(Number(r.visit_id), r);
  }
  return map;
}

function hostFromUrl(url) {
  try {
    return new URL(url).hostname || url;
  } catch (_) {
    return url || "";
  }
}

function hashColor(s) {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  const hue = Math.abs(h) % 360;
  return `hsl(${hue}, 60%, 55%)`;
}

function timeMs(rec) {
  if (rec.visit_time_utc_iso) {
    const t = Date.parse(rec.visit_time_utc_iso);
    if (!Number.isNaN(t)) return t;
  }
  return null;
}

function buildRoots(records, byId) {
  const rootMap = new Map();
  const roots = new Map();
  function getRoot(id) {
    if (rootMap.has(id)) return rootMap.get(id);
    let current = id;
    const seen = new Set();
    while (true) {
      if (seen.has(current)) break;
      seen.add(current);
      const rec = byId.get(current);
      if (!rec) break;
      const from = Number(rec.from_visit || 0);
      if (!from || !byId.has(from)) break;
      current = from;
    }
    rootMap.set(id, current);
    return current;
  }

  for (const r of records) {
    const id = Number(r.visit_id);
    const root = getRoot(id);
    if (!roots.has(root)) roots.set(root, []);
    roots.get(root).push(id);
  }

  const rootList = Array.from(roots.keys());
  rootList.sort((a, b) => {
    const ra = byId.get(a);
    const rb = byId.get(b);
    return (timeMs(ra) || 0) - (timeMs(rb) || 0);
  });

  return { roots, rootList, rootMap };
}

function render() {
  const byId = byIdMap(RECORDS);
  const { roots, rootList, rootMap } = buildRoots(RECORDS, byId);
  const laneHeight = 80;
  const marginLeft = 80;
  const marginTop = 20;
  const scale = Number(scaleInput.value);

  const times = RECORDS.map(timeMs).filter(t => t !== null);
  const minTime = Math.min(...times);
  const maxTime = Math.max(...times);
  const spanMinutes = Math.max(1, (maxTime - minTime) / 60000);
  const width = Math.max(800, marginLeft + spanMinutes * scale + 200);
  const height = marginTop + rootList.length * laneHeight + 40;

  svg.setAttribute("width", width);
  svg.setAttribute("height", height);
  svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
  svg.innerHTML = "";

  summaryEl.textContent = `${RECORDS.length} visits, ${rootList.length} lanes`;

  // lane labels
  rootList.forEach((root, idx) => {
    const rec = byId.get(root);
    const label = rec ? hostFromUrl(rec.url || "") : `root ${root}`;
    const y = marginTop + idx * laneHeight + laneHeight / 2;
    const text = document.createElementNS("http://www.w3.org/2000/svg", "text");
    text.setAttribute("x", 12);
    text.setAttribute("y", y + 4);
    text.setAttribute("class", "lane-label");
    text.textContent = label.slice(0, 18);
    svg.appendChild(text);
  });

  // edges
  for (const rec of RECORDS) {
    const from = Number(rec.from_visit || 0);
    if (!from || !byId.has(from)) continue;
    const src = byId.get(from);
    const srcLane = rootList.indexOf(rootMap.get(from));
    const dstLane = rootList.indexOf(rootMap.get(Number(rec.visit_id)));
    const srcTime = timeMs(src);
    const dstTime = timeMs(rec);
    if (srcTime === null || dstTime === null) continue;
    const x1 = marginLeft + ((srcTime - minTime) / 60000) * scale;
    const x2 = marginLeft + ((dstTime - minTime) / 60000) * scale;
    const y1 = marginTop + srcLane * laneHeight + laneHeight / 2;
    const y2 = marginTop + dstLane * laneHeight + laneHeight / 2;
    const line = document.createElementNS("http://www.w3.org/2000/svg", "line");
    line.setAttribute("x1", x1);
    line.setAttribute("y1", y1);
    line.setAttribute("x2", x2);
    line.setAttribute("y2", y2);
    line.setAttribute("class", "edge");
    svg.appendChild(line);
  }

  // nodes
  for (const rec of RECORDS) {
    const id = Number(rec.visit_id);
    const lane = rootList.indexOf(rootMap.get(id));
    const t = timeMs(rec);
    if (t === null || lane < 0) continue;
    const x = marginLeft + ((t - minTime) / 60000) * scale;
    const y = marginTop + lane * laneHeight + laneHeight / 2;
    const host = hostFromUrl(rec.url || "");
    const color = hashColor(host);

    const circle = document.createElementNS("http://www.w3.org/2000/svg", "circle");
    circle.setAttribute("cx", x);
    circle.setAttribute("cy", y);
    circle.setAttribute("r", 6);
    circle.setAttribute("fill", color);
    circle.setAttribute("class", "node");
    if (rec.hidden) {
      circle.setAttribute("stroke", "#999999");
      circle.setAttribute("stroke-dasharray", "2 2");
    } else {
      circle.setAttribute("stroke", "#121823");
    }

    circle.addEventListener("click", () => {
      detailJson.textContent = JSON.stringify(rec, null, 2);
    });

    const title = document.createElementNS("http://www.w3.org/2000/svg", "title");
    title.textContent = `${host}\n${rec.title || ""}\n${rec.visit_time_utc_iso || ""}`;
    circle.appendChild(title);
    svg.appendChild(circle);
  }
}

scaleInput.addEventListener("input", render);
render();
</script>
</body>
</html>
"""
    html = template.replace("__DATA_JSON__", data_json)
    with open(out_path, "w", encoding="utf-8") as fp:
        fp.write(html)


def write_graph(records: List[Dict[str, Any]], fmt: str, out_path: Optional[str]) -> None:
    fmt = fmt.lower()
    if fmt not in {"svg", "png", "dot", "html"}:
        raise ValueError("--graph must be one of: svg, png, dot, html")

    default_name = f"history_graph.{fmt}"
    target = out_path or default_name

    if fmt == "html":
        write_graph_html(records, target)
        return

    dot = build_dot(records)

    if fmt == "dot":
        with open(target, "w", encoding="utf-8") as fp:
            fp.write(dot)
        return

    dot_path = Path(tempfile.mkdtemp(prefix="brave-history-graph-")) / "graph.dot"
    dot_path.write_text(dot, encoding="utf-8")

    try:
        subprocess.run(["dot", f"-T{fmt}", "-o", target, str(dot_path)], check=True)
    except FileNotFoundError as e:
        raise RuntimeError("graphviz 'dot' not found; install graphviz or use --graph dot") from e
    finally:
        try:
            shutil.rmtree(dot_path.parent, ignore_errors=True)
        except Exception:
            pass


def run_query(db_copy: Path, sql: str, params: List[Any], time_keys: Iterable[str]) -> List[Dict[str, Any]]:
    conn = sqlite3.connect(db_copy)
    try:
        cur = conn.cursor()
        cur.execute(sql, params)
        rows = cur.fetchall()
        dicts = rows_to_dicts(cur, rows)
    finally:
        conn.close()

    for d in dicts:
        normalize_time_fields(d, time_keys)
    return dicts


# ----------------------------
# Main
# ----------------------------

def main() -> int:
    p = argparse.ArgumentParser(description="Dump Brave history from the History SQLite DB.")
    p.add_argument("--history", help="Path to History DB file (overrides auto-detect).")
    p.add_argument("--profile", help='Profile folder name, e.g. "Default" or "Profile 2" (auto-detect if omitted).')
    p.add_argument("--limit", type=int, default=0, help="Limit rows (0 = no limit).")
    p.add_argument("--since-days", type=int, help="Only include rows since N days ago (UTC).")
    p.add_argument("--since-time", help='Only include rows after this ISO time (e.g. "2026-02-01T00:00:00Z").')
    p.add_argument("--until-time", help='Only include rows before this ISO time (e.g. "2026-02-03T23:59:59Z").')
    p.add_argument("--out", help="Write to file instead of stdout.")
    p.add_argument("--csv", action="store_true", help="Output CSV (default JSONL).")

    sub = p.add_subparsers(dest="cmd", required=True)

    sub.add_parser("visits", help="Dump raw visits table.")

    sub_joined = sub.add_parser("joined", help="Dump visits joined with urls.")
    sub_joined.add_argument("--url-like", help='SQL LIKE filter for URL, e.g. "%%github.com%%".')
    sub_joined.add_argument("--graph", help='Write visit graph (svg/png/dot). Requires graphviz for svg/png.')
    sub_joined.add_argument("--graph-out", help="Path for --graph output (default: history_graph.<ext>).")

    sub_urls = sub.add_parser("urls", help="Dump urls table (unique URL entries).")
    sub_urls.add_argument("--url-like", help='SQL LIKE filter for URL, e.g. "%%github.com%%".')

    args = p.parse_args()

    # Resolve DB path
    if args.history:
        db_path = Path(args.history).expanduser()
        if not db_path.exists():
            raise FileNotFoundError(f"History DB not found: {db_path}")
    else:
        db_path = history_db_path(args.profile)

    db_copy = copy_db_for_reading(db_path)

    try:
        limit = f"LIMIT {args.limit}" if args.limit and args.limit > 0 else ""

        if args.cmd == "visits":
            where, params = build_where_and_params(
                since_days=args.since_days,
                since_time=args.since_time,
                until_time=args.until_time,
                url_like=None,
                time_expr="visits.visit_time",
                url_expr=None,
            )
            sql = VISITS_SQL.format(where=where, limit=limit)
            records = run_query(db_copy, sql, params, time_keys=["visit_time"])

        elif args.cmd == "joined":
            where, params = build_where_and_params(
                since_days=args.since_days,
                since_time=args.since_time,
                until_time=args.until_time,
                url_like=args.url_like,
                time_expr="v.visit_time",
                url_expr="u.url",
            )
            sql = JOINED_SQL.format(where=where, limit=limit)
            records = run_query(db_copy, sql, params, time_keys=["visit_time", "last_visit_time"])
            if args.graph:
                write_graph(records, args.graph, args.graph_out)

        elif args.cmd == "urls":
            where, params = build_where_and_params(
                since_days=args.since_days,
                since_time=args.since_time,
                until_time=args.until_time,
                url_like=getattr(args, "url_like", None),
                time_expr="urls.last_visit_time",
                url_expr="urls.url",
            )
            sql = URLS_SQL.format(where=where, limit=limit)
            records = run_query(db_copy, sql, params, time_keys=["last_visit_time"])

        else:
            raise ValueError(f"Unknown command: {args.cmd}")

        out_fp = open(args.out, "w", encoding="utf-8", newline="") if args.out else sys.stdout
        try:
            if args.csv:
                write_csv(records, out_fp)
            else:
                write_jsonl(records, out_fp)
        finally:
            if args.out:
                out_fp.close()

    finally:
        try:
            shutil.rmtree(db_copy.parent, ignore_errors=True)
        except Exception:
            pass

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
