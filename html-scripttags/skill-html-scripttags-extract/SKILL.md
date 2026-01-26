---
name: skill-html-scripttags-extract
description: Extract script tags from HTML with optional attribute filters using the bundled html-scripttags-extract.py tool.
---

# HTML Script Tag Extract

Use this skill to extract `<script>...</script>` tags from HTML, optionally filtering by attributes.

## Commands

Extract all script tags:

```bash
python3 skills/skill-html-scripttags-extract/scripts/html-scripttags-extract.py --file path/to/page.html
```

Filter by type and remove script tags:

```bash
python3 skills/skill-html-scripttags-extract/scripts/html-scripttags-extract.py \
  --type "application/json" \
  --clean \
  --file path/to/page.html
```

Filter by id and write output to a file:

```bash
python3 skills/skill-html-scripttags-extract/scripts/html-scripttags-extract.py \
  --id "airgap-GPP" \
  --out extracted.txt \
  --file path/to/page.html
```

Pipe HTML via stdin:

```bash
cat path/to/page.html | python3 skills/skill-html-scripttags-extract/scripts/html-scripttags-extract.py --type "foobar"
```

## Notes

- Filters are exact matches for `type`, `id`, and `src`.
- Use `--clean` to output only script contents (no surrounding tags).
