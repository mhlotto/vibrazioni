---
name: skill-ietf-draft-extract
description: List table-of-contents or extract specific sections from IETF Internet-Draft/RFC plain-text files using the bundled ietf-draft-extract.py script. Use for --toc listings or section extraction with optional --clean.
---

# IETF Draft Extract

Use this skill when a user wants to list a draft/RFC table of contents or extract one or more sections from an IETF plain-text file.

## Commands

List TOC:

```bash
python3 skill-ietf-draft-extract/scripts/ietf-draft-extract.py --toc path/to/draft.txt
```

Full clean of entire draft (no section query):

```bash
python3 skill-ietf-draft-extract/scripts/ietf-draft-extract.py --full-clean path/to/draft.txt
```

Extract section(s) with boilerplate removed (always use --clean):

```bash
python3 skill-ietf-draft-extract/scripts/ietf-draft-extract.py --clean path/to/draft.txt "3.2,4.1"
```

Extract by title substring with clean output:

```bash
python3 skill-ietf-draft-extract/scripts/ietf-draft-extract.py --clean path/to/draft.txt "security considerations"
```

## Notes

- Queries accept section numbers ("3.2") or title substrings.
- Multiple queries are comma-separated.
