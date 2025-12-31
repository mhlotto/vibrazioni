---
name: skill-govulncheck-json-extract
description: Extract and format findings from govulncheck JSON output using the bundled govulncheck-json-extract.bash script. Use for CLI-friendly summaries, filtering, grouping, sorting, and stats.
---

# Govulncheck JSON Extract

Use this skill to summarize and filter govulncheck JSON output without dumping raw JSON.

## Commands

Default one-line summaries:

```bash
bash skill-govulncheck-json-extract/scripts/govulncheck-json-extract.bash path/to/report.json
```

Filter and sort:

```bash
bash skill-govulncheck-json-extract/scripts/govulncheck-json-extract.bash \
  --module golang.org/x/crypto \
  --sort osv \
  path/to/report.json
```

Group by module and customize format:

```bash
bash skill-govulncheck-json-extract/scripts/govulncheck-json-extract.bash \
  --group module \
  --format "{module} {osv} {summary}" \
  path/to/report.json
```

Stats only:

```bash
bash skill-govulncheck-json-extract/scripts/govulncheck-json-extract.bash --stats path/to/report.json
```

## Notes

- You can pipe JSON via stdin instead of passing a file path.
- Valid group/sort fields: osv, module, package, fixed_version, summary.
