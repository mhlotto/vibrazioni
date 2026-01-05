#!/usr/bin/python3

import argparse
import re
import secrets
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import List, Optional

GROUP_RE = re.compile(r"^##\s+(?:\[(GP[0-9a-fA-F]{3})\]\s+)?(.*\S)\s*$")
TASK_RE = re.compile(
    r"^(\s*)-\s+\[( |x|X)\]\s+(?:\[(T[0-9a-fA-F]{5})\]\s+)?(.*\S)\s*$"
)


@dataclass
class Group:
    id: Optional[str]
    name: str
    start: int
    end: int


@dataclass
class Task:
    id: Optional[str]
    text: str
    checked: bool
    indent: int
    index: int
    parent_index: Optional[int]


def read_lines(path: Path) -> List[str]:
    return path.read_text(encoding="utf-8").splitlines(keepends=True)


def write_lines(path: Path, lines: List[str]) -> None:
    path.write_text("".join(lines), encoding="utf-8")


def parse_groups(lines: List[str]) -> List[Group]:
    starts = []
    for i, line in enumerate(lines):
        m = GROUP_RE.match(line.rstrip("\n"))
        if not m:
            continue
        gid = m.group(1)
        name = m.group(2).strip()
        starts.append((i, gid.lower() if gid else None, name))

    groups = []
    for idx, (start, gid, name) in enumerate(starts):
        end = starts[idx + 1][0] if idx + 1 < len(starts) else len(lines)
        groups.append(Group(id=gid, name=name, start=start, end=end))
    return groups


def parse_tasks(lines: List[str], group: Group) -> List[Task]:
    tasks = []
    current_parent = None
    for i in range(group.start + 1, group.end):
        m = TASK_RE.match(lines[i].rstrip("\n"))
        if not m:
            continue
        indent = len(m.group(1))
        checked = m.group(2).lower() == "x"
        tid = m.group(3)
        tid = tid.lower() if tid else None
        text = m.group(4).strip()
        parent_index = None
        if indent == 0:
            current_parent = i
        elif current_parent is not None:
            parent_index = current_parent
        tasks.append(Task(id=tid, text=text, checked=checked, indent=indent, index=i, parent_index=parent_index))
    return tasks


def find_unique(items, selector: str, kind: str):
    needle = selector.strip()
    if not needle:
        raise SystemExit(f"{kind} selector is empty")

    needle_l = needle.lower()
    id_matches = [it for it in items if it.id and it.id.lower() == needle_l]
    if id_matches:
        if len(id_matches) == 1:
            return id_matches[0]
        raise SystemExit(f"Ambiguous {kind} selector: {selector}")

    text_matches = [it for it in items if needle_l in it.text.lower()]
    if not text_matches:
        raise SystemExit(f"No {kind} matched: {selector}")
    if len(text_matches) > 1:
        sample = ", ".join(
            f"{it.id or '-'}:{it.text}" for it in text_matches[:5]
        )
        raise SystemExit(f"Multiple {kind} matches for '{selector}': {sample}")
    return text_matches[0]


def find_group(groups: List[Group], selector: str) -> Group:
    needle = selector.strip()
    if not needle:
        raise SystemExit("group selector is empty")

    needle_l = needle.lower()
    id_matches = [g for g in groups if g.id and g.id.lower() == needle_l]
    if id_matches:
        if len(id_matches) == 1:
            return id_matches[0]
        raise SystemExit(f"Ambiguous group selector: {selector}")

    text_matches = [g for g in groups if needle_l in g.name.lower()]
    if not text_matches:
        raise SystemExit(f"No group matched: {selector}")
    if len(text_matches) > 1:
        sample = ", ".join(
            f"{g.id or '-'}:{g.name}" for g in text_matches[:5]
        )
        raise SystemExit(f"Multiple group matches for '{selector}': {sample}")
    return text_matches[0]


def collect_existing_ids(lines: List[str]):
    group_ids = set()
    task_ids = set()
    for line in lines:
        gm = GROUP_RE.match(line.rstrip("\n"))
        if gm and gm.group(1):
            group_ids.add(gm.group(1).lower())
        tm = TASK_RE.match(line.rstrip("\n"))
        if tm and tm.group(3):
            task_ids.add(tm.group(3).lower())
    return group_ids, task_ids


def new_id(prefix: str, length: int, existing: set) -> str:
    while True:
        raw = secrets.token_hex((length + 1) // 2)[:length]
        candidate = f"{prefix}{raw}"
        if candidate.lower() not in existing:
            existing.add(candidate.lower())
            return candidate


def ensure_newline(line: str) -> str:
    return line if line.endswith("\n") else line + "\n"


def init_plan(args) -> None:
    if not re.fullmatch(r"\d{4}", args.id):
        raise SystemExit("--id must be 4 digits")
    dest_dir = Path(args.dir)
    dest_dir.mkdir(parents=True, exist_ok=True)
    path = dest_dir / f"PLAN{args.id}.md"
    if path.exists():
        raise SystemExit(f"File already exists: {path}")

    title = args.title.strip()
    goal = args.goal.strip()
    if not title:
        raise SystemExit("--title is required")
    if not goal:
        raise SystemExit("--goal is required")

    lines = [
        f"# Plan {args.id}: {title}\n",
        "\n",
        f"Goal: {goal}\n",
        "\n",
    ]
    write_lines(path, lines)
    print(path)


def list_groups(args) -> None:
    lines = read_lines(Path(args.file))
    groups = parse_groups(lines)
    for g in groups:
        print(lines[g.start].rstrip("\n"))


def add_group(args) -> None:
    path = Path(args.file)
    lines = read_lines(path)
    groups = parse_groups(lines)
    group_ids, _ = collect_existing_ids(lines)
    gid = new_id("GP", 3, group_ids)

    name = args.name.strip()
    if not name:
        raise SystemExit("group name is empty")

    insert_idx = len(lines)
    while insert_idx > 0 and lines[insert_idx - 1].strip() == "":
        insert_idx -= 1
    if insert_idx > 0 and lines[insert_idx - 1].strip() != "":
        lines.insert(insert_idx, "\n")
        insert_idx += 1

    lines.insert(insert_idx, ensure_newline(f"## [{gid}] {name}"))
    write_lines(path, lines)
    print(gid)


def rename_group(args) -> None:
    path = Path(args.file)
    lines = read_lines(path)
    groups = parse_groups(lines)
    group = find_group(groups, args.group)
    new_name = args.name.strip()
    if not new_name:
        raise SystemExit("new group name is empty")

    old_line = lines[group.start].rstrip("\n")
    m = GROUP_RE.match(old_line)
    if not m:
        raise SystemExit("group heading not found")
    gid = m.group(1)
    prefix = f"## [{gid}] " if gid else "## "
    lines[group.start] = ensure_newline(prefix + new_name)
    write_lines(path, lines)


def list_tasks(args) -> None:
    lines = read_lines(Path(args.file))
    groups = parse_groups(lines)
    target_groups = groups
    if args.group:
        target_groups = [find_group(groups, args.group)]

    for group in target_groups:
        tasks = parse_tasks(lines, group)
        for t in tasks:
            if not args.ws and t.indent != 0:
                continue
            if args.search:
                needle = args.search.lower()
                if needle not in t.text.lower() and (t.id or "") != needle:
                    continue
            print(lines[t.index].rstrip("\n"))


def add_task(args) -> None:
    path = Path(args.file)
    lines = read_lines(path)
    groups = parse_groups(lines)
    group = find_group(groups, args.group)
    _, task_ids = collect_existing_ids(lines)
    tid = new_id("T", 5, task_ids)

    text = args.text.strip()
    if not text:
        raise SystemExit("task text is empty")

    insert_idx = group.end
    while insert_idx > group.start + 1 and lines[insert_idx - 1].strip() == "":
        insert_idx -= 1

    line = f"- [ ] [{tid}] {text}"
    lines.insert(insert_idx, ensure_newline(line))
    write_lines(path, lines)
    print(tid)


def rename_task(args) -> None:
    path = Path(args.file)
    lines = read_lines(path)
    groups = parse_groups(lines)
    group = find_group(groups, args.group)
    tasks = [t for t in parse_tasks(lines, group) if t.indent == 0]
    task = find_unique(tasks, args.task, "task")
    new_name = args.name.strip()
    if not new_name:
        raise SystemExit("new task name is empty")

    line = lines[task.index].rstrip("\n")
    m = TASK_RE.match(line)
    if not m:
        raise SystemExit("task line not found")
    indent = m.group(1)
    box = m.group(2)
    tid = m.group(3)
    id_part = f"[{tid}] " if tid else ""
    lines[task.index] = ensure_newline(f"{indent}- [{box}] {id_part}{new_name}")
    write_lines(path, lines)


def task_block_bounds(lines: List[str], group: Group, task: Task, tasks: List[Task]):
    if task.indent != 0:
        raise SystemExit("expected a top-level task")
    next_top = None
    for t in tasks:
        if t.indent == 0 and t.index > task.index:
            next_top = t.index
            break
    start = task.index
    end = next_top if next_top is not None else group.end
    return start, end


def move_task(args) -> None:
    path = Path(args.file)
    lines = read_lines(path)
    groups = parse_groups(lines)
    from_group = find_group(groups, args.from_group)
    to_group = find_group(groups, args.to_group)

    tasks_from = [t for t in parse_tasks(lines, from_group) if t.indent == 0]
    task = find_unique(tasks_from, args.task, "task")
    block_start, block_end = task_block_bounds(lines, from_group, task, tasks_from)
    block_lines = lines[block_start:block_end]

    del lines[block_start:block_end]

    groups = parse_groups(lines)
    to_group = find_group(groups, args.to_group)
    insert_idx = to_group.end
    while insert_idx > to_group.start + 1 and lines[insert_idx - 1].strip() == "":
        insert_idx -= 1

    for i, line in enumerate(block_lines):
        lines.insert(insert_idx + i, line)

    write_lines(path, lines)


def toggle_task(args) -> None:
    path = Path(args.file)
    lines = read_lines(path)
    groups = parse_groups(lines)
    group = find_group(groups, args.group)
    tasks = [t for t in parse_tasks(lines, group) if t.indent == 0]
    task = find_unique(tasks, args.task, "task")

    line = lines[task.index].rstrip("\n")
    m = TASK_RE.match(line)
    if not m:
        raise SystemExit("task line not found")
    indent = m.group(1)
    box = m.group(2)
    tid = m.group(3)
    text = m.group(4)
    new_box = " " if box.lower() == "x" else "x"
    id_part = f"[{tid}] " if tid else ""
    lines[task.index] = ensure_newline(f"{indent}- [{new_box}] {id_part}{text}")
    write_lines(path, lines)


def add_subtask(args) -> None:
    path = Path(args.file)
    lines = read_lines(path)
    groups = parse_groups(lines)
    group = find_group(groups, args.group)
    tasks = parse_tasks(lines, group)
    parents = [t for t in tasks if t.indent == 0]
    parent = find_unique(parents, args.task, "task")

    _, task_ids = collect_existing_ids(lines)
    tid = new_id("T", 5, task_ids)

    text = args.text.strip()
    if not text:
        raise SystemExit("subtask text is empty")

    block_start, block_end = task_block_bounds(lines, group, parent, parents)
    insert_idx = block_end
    while insert_idx > block_start + 1 and lines[insert_idx - 1].strip() == "":
        insert_idx -= 1

    line = f"  - [ ] [{tid}] {text}"
    lines.insert(insert_idx, ensure_newline(line))
    write_lines(path, lines)
    print(tid)


def list_subtasks(args) -> None:
    lines = read_lines(Path(args.file))
    groups = parse_groups(lines)
    group = find_group(groups, args.group)
    tasks = parse_tasks(lines, group)
    parents = [t for t in tasks if t.indent == 0]
    parent = find_unique(parents, args.task, "task")

    for t in tasks:
        if t.parent_index == parent.index:
            print(lines[t.index].rstrip("\n"))


def rename_subtask(args) -> None:
    path = Path(args.file)
    lines = read_lines(path)
    groups = parse_groups(lines)
    group = find_group(groups, args.group)
    tasks = parse_tasks(lines, group)
    parents = [t for t in tasks if t.indent == 0]
    parent = find_unique(parents, args.task, "task")

    subtasks = [t for t in tasks if t.parent_index == parent.index]
    subtask = find_unique(subtasks, args.subtask, "subtask")
    new_name = args.name.strip()
    if not new_name:
        raise SystemExit("new subtask name is empty")

    line = lines[subtask.index].rstrip("\n")
    m = TASK_RE.match(line)
    if not m:
        raise SystemExit("subtask line not found")
    indent = m.group(1)
    box = m.group(2)
    tid = m.group(3)
    id_part = f"[{tid}] " if tid else ""
    lines[subtask.index] = ensure_newline(f"{indent}- [{box}] {id_part}{new_name}")
    write_lines(path, lines)


def toggle_subtask(args) -> None:
    path = Path(args.file)
    lines = read_lines(path)
    groups = parse_groups(lines)
    group = find_group(groups, args.group)
    tasks = parse_tasks(lines, group)
    parents = [t for t in tasks if t.indent == 0]
    parent = find_unique(parents, args.task, "task")

    subtasks = [t for t in tasks if t.parent_index == parent.index]
    subtask = find_unique(subtasks, args.subtask, "subtask")

    line = lines[subtask.index].rstrip("\n")
    m = TASK_RE.match(line)
    if not m:
        raise SystemExit("subtask line not found")
    indent = m.group(1)
    box = m.group(2)
    tid = m.group(3)
    text = m.group(4)
    new_box = " " if box.lower() == "x" else "x"
    id_part = f"[{tid}] " if tid else ""
    lines[subtask.index] = ensure_newline(f"{indent}- [{new_box}] {id_part}{text}")
    write_lines(path, lines)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Manage PLAN####.md files.")
    sub = parser.add_subparsers(dest="command", required=True)

    init_p = sub.add_parser("init", help="Initialize a plan file")
    init_p.add_argument("--id", required=True, help="4-digit plan id")
    init_p.add_argument("--title", required=True, help="Short plan title")
    init_p.add_argument("--goal", required=True, help="Plan goal")
    init_p.add_argument("--dir", default=".", help="Destination directory")
    init_p.set_defaults(func=init_plan)

    groups_p = sub.add_parser("groups", help="Manage groups")
    groups_sub = groups_p.add_subparsers(dest="groups_cmd", required=True)

    groups_list = groups_sub.add_parser("list", help="List groups")
    groups_list.add_argument("--file", required=True, help="PLAN file path")
    groups_list.set_defaults(func=list_groups)

    groups_add = groups_sub.add_parser("add", help="Add a group")
    groups_add.add_argument("--file", required=True, help="PLAN file path")
    groups_add.add_argument("name", help="Group name")
    groups_add.set_defaults(func=add_group)

    groups_rename = groups_sub.add_parser("rename", help="Rename a group")
    groups_rename.add_argument("--file", required=True, help="PLAN file path")
    groups_rename.add_argument("--group", required=True, help="Group id or name")
    groups_rename.add_argument("--name", required=True, help="New group name")
    groups_rename.set_defaults(func=rename_group)

    tasks_p = sub.add_parser("tasks", help="Manage tasks")
    tasks_sub = tasks_p.add_subparsers(dest="tasks_cmd", required=True)

    tasks_list = tasks_sub.add_parser("list", help="List tasks in a group")
    tasks_list.add_argument("--file", required=True, help="PLAN file path")
    tasks_list.add_argument("--group", help="Group id or name")
    tasks_list.add_argument("--search", help="Filter by task text")
    tasks_list.add_argument("--ws", action="store_true", help="Include subtasks")
    tasks_list.set_defaults(func=list_tasks)

    tasks_add = tasks_sub.add_parser("add", help="Add a task")
    tasks_add.add_argument("--file", required=True, help="PLAN file path")
    tasks_add.add_argument("--group", required=True, help="Group id or name")
    tasks_add.add_argument("text", help="Task text")
    tasks_add.set_defaults(func=add_task)

    tasks_rename = tasks_sub.add_parser("rename", help="Rename a task")
    tasks_rename.add_argument("--file", required=True, help="PLAN file path")
    tasks_rename.add_argument("--group", required=True, help="Group id or name")
    tasks_rename.add_argument("--task", required=True, help="Task id or text")
    tasks_rename.add_argument("--name", required=True, help="New task text")
    tasks_rename.set_defaults(func=rename_task)

    tasks_move = tasks_sub.add_parser("move", help="Move a task to another group")
    tasks_move.add_argument("--file", required=True, help="PLAN file path")
    tasks_move.add_argument("--task", required=True, help="Task id or text")
    tasks_move.add_argument("--from", dest="from_group", required=True, help="Source group id or name")
    tasks_move.add_argument("--to", dest="to_group", required=True, help="Destination group id or name")
    tasks_move.set_defaults(func=move_task)

    tasks_x = tasks_sub.add_parser("x", help="Toggle task completion")
    tasks_x.add_argument("--file", required=True, help="PLAN file path")
    tasks_x.add_argument("--group", required=True, help="Group id or name")
    tasks_x.add_argument("--task", required=True, help="Task id or text")
    tasks_x.set_defaults(func=toggle_task)

    subtasks_p = sub.add_parser("subtasks", help="Manage subtasks")
    subtasks_sub = subtasks_p.add_subparsers(dest="subtasks_cmd", required=True)

    subtasks_list = subtasks_sub.add_parser("list", help="List subtasks for a task")
    subtasks_list.add_argument("--file", required=True, help="PLAN file path")
    subtasks_list.add_argument("--group", required=True, help="Group id or name")
    subtasks_list.add_argument("--task", required=True, help="Task id or text")
    subtasks_list.set_defaults(func=list_subtasks)

    subtasks_add = subtasks_sub.add_parser("add", help="Add a subtask")
    subtasks_add.add_argument("--file", required=True, help="PLAN file path")
    subtasks_add.add_argument("--group", required=True, help="Group id or name")
    subtasks_add.add_argument("--task", required=True, help="Task id or text")
    subtasks_add.add_argument("text", help="Subtask text")
    subtasks_add.set_defaults(func=add_subtask)

    subtasks_rename = subtasks_sub.add_parser("rename", help="Rename a subtask")
    subtasks_rename.add_argument("--file", required=True, help="PLAN file path")
    subtasks_rename.add_argument("--group", required=True, help="Group id or name")
    subtasks_rename.add_argument("--task", required=True, help="Task id or text")
    subtasks_rename.add_argument("--subtask", required=True, help="Subtask id or text")
    subtasks_rename.add_argument("--name", required=True, help="New subtask text")
    subtasks_rename.set_defaults(func=rename_subtask)

    subtasks_x = subtasks_sub.add_parser("x", help="Toggle subtask completion")
    subtasks_x.add_argument("--file", required=True, help="PLAN file path")
    subtasks_x.add_argument("--group", required=True, help="Group id or name")
    subtasks_x.add_argument("--task", required=True, help="Task id or text")
    subtasks_x.add_argument("--subtask", required=True, help="Subtask id or text")
    subtasks_x.set_defaults(func=toggle_subtask)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    args.func(args)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
