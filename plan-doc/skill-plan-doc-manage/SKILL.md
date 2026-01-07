---
name: skill-plan-doc-manage
description: Create and manage PLAN####.md files using the bundled plan-doc-manage.py script. Use for initializing plans, listing groups, and adding/renaming/moving/toggling tasks and subtasks.
---

# Plan Doc Manage

Use this skill to manage PLAN####.md files with groupings, tasks, and subtasks.

## Commands

Initialize a plan file:

```bash
python3 skill-plan-doc-manage/scripts/plan-doc-manage.py init \
  --id 0002 \
  --title "Focused Follow On Work" \
  --goal "Keep a short list of active, high value items." \
  --dir .
```

List groups:

```bash
python3 skill-plan-doc-manage/scripts/plan-doc-manage.py \
  groups list --file PLAN0002.md
```

Add a group (auto IDs like GPabc):

```bash
python3 skill-plan-doc-manage/scripts/plan-doc-manage.py \
  groups add --file PLAN0002.md "Top Priorities"
```

Add a task (auto IDs like Tabc12):

```bash
python3 skill-plan-doc-manage/scripts/plan-doc-manage.py \
  tasks add --file PLAN0002.md --group GPabc "Finish demo runthrough"
```

List tasks in a group:

```bash
python3 skill-plan-doc-manage/scripts/plan-doc-manage.py \
  tasks list --file PLAN0002.md --group GPabc
```

Toggle a task complete/incomplete:

```bash
python3 skill-plan-doc-manage/scripts/plan-doc-manage.py \
  tasks x --file PLAN0002.md --group GPabc --task Tabc12
```

Add a subtask:

```bash
python3 skill-plan-doc-manage/scripts/plan-doc-manage.py \
  subtasks add --file PLAN0002.md --group GPabc --task Tabc12 "Write fixtures"
```

## Notes

- Group selectors accept IDs (GPxxx) or name substrings.
- Task and subtask selectors accept IDs (Txxxxx) or text substrings.
- The tool preserves existing Markdown structure and only edits matching lines.
