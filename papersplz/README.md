# papersplz

`papersplz` is a local-first command-line tool for cataloging academic papers
and related documents. The v1 design is documented in `V1.md`.

The CLI recognizes the planned v1 commands and resolves a catalog from
`--home PATH` or `PAPERSPLZ_HOME`. Catalog initialization and document import
from local files or direct HTTP/HTTPS URLs are implemented. Catalog inspection
is available through `show`, `list`, and `path`. Paper removal is
confirmation-protected, tags can be added, removed, and listed, and each paper
can have a single review that can be set, shown, edited, or removed. Remaining
catalog operations are not yet implemented.

```sh
go run ./cmd/papersplz help
go run ./cmd/papersplz init /path/to/catalog --name "My Papers"
go run ./cmd/papersplz --home /path/to/catalog add paper.pdf --title "A Paper"
go run ./cmd/papersplz --home /path/to/catalog add https://example.org/paper.pdf --title "A Remote Paper"
go run ./cmd/papersplz --home /path/to/catalog list
go run ./cmd/papersplz --home /path/to/catalog show PAPER_ID --json
go run ./cmd/papersplz --home /path/to/catalog path PAPER_ID
go run ./cmd/papersplz --home /path/to/catalog remove PAPER_ID --yes
go run ./cmd/papersplz --home /path/to/catalog tag add PAPER_ID to-read topology
go run ./cmd/papersplz --home /path/to/catalog tag list PAPER_ID --json
go run ./cmd/papersplz --home /path/to/catalog review set PAPER_ID "Review text"
go run ./cmd/papersplz --home /path/to/catalog review show PAPER_ID --json
```
