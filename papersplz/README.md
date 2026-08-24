# papersplz

`papersplz` is a local-first command-line tool for cataloging academic papers
and related documents. The v1 design is documented in `V1.md`.

The CLI scaffold currently recognizes the planned v1 commands and resolves a
catalog from `--home PATH` or `PAPERSPLZ_HOME`. Catalog operations are not yet
implemented.

```sh
go run ./cmd/papersplz help
go run ./cmd/papersplz --home /path/to/catalog list
```
