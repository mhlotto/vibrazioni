---
name: skill-gosec-json-extract
description: Extract and format findings from gosec JSON reports using the bundled gosec-json-extract.bash script. Use for CLI-friendly summaries, filtering, grouping, sorting, and stats.
---

# Gosec JSON Extract

Use this skill to summarize and filter gosec JSON reports without dumping raw JSON.

## Commands

Default one-line summaries:

```bash
bash skills/skill-gosec-json-extract/scripts/gosec-json-extract.bash path/to/report.json
```

Filter and sort:

```bash
bash skills/skill-gosec-json-extract/scripts/gosec-json-extract.bash --severity HIGH --sort file path/to/report.json
```

Group by rule and customize format:

```bash
bash skills/skill-gosec-json-extract/scripts/gosec-json-extract.bash \
  --group rule_id \
  --format "{rule_id} {location} {details}" \
  path/to/report.json
```

Stats only:

```bash
bash skills/skill-gosec-json-extract/scripts/gosec-json-extract.bash --stats path/to/report.json
```

## Notes

- You can pipe JSON via stdin instead of passing a file path.
- Valid group or sort fields: severity, confidence, cwe, rule_id, file, line.
