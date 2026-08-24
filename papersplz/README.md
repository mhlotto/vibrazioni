![papersplz logo](assets/papersplz-logo.png)

# papersplz

`papersplz` is a local-first CLI for cataloging academic papers and related
documents. It stores ordinary documents and human-readable JSON in a
self-contained catalog—there is no database, server, or platform-specific
runtime service. The short command name `pplz` runs the same program.

The v1 design is documented in [V1.md](V1.md). The program supports macOS,
Linux, and FreeBSD and requires Go 1.21 or newer to build.

## Build and install

From this directory:

```sh
make build     # creates bin/papersplz and the bin/pplz symlink
make check     # tests, vets, builds, and verifies the alias
```

`make test` and `make vet` run the individual checks. Make is only a
convenience; the equivalent build is:

```sh
mkdir -p bin
go build -o bin/papersplz ./cmd/papersplz
ln -sf papersplz bin/pplz
```

The Makefile uses standard tools available on macOS, Linux, and FreeBSD.
Install under `/usr/local/bin`, or choose an unprivileged prefix on your PATH:

```sh
sudo make install
make install PREFIX="$HOME/.local"
```

Packagers may override `BINDIR`, `DESTDIR`, and `BUILD_DIR`; for example,
`make install DESTDIR=/tmp/package-root PREFIX=/usr` stages files beneath
`/tmp/package-root/usr/bin`.

## Choose and create a catalog

Select a catalog with `PAPERSPLZ_HOME`:

```sh
export PAPERSPLZ_HOME="$HOME/papers/my-catalog"
pplz list
```

or put the global `--home` option before the command:

```sh
pplz --home "$HOME/papers/another-catalog" list
```

`--home` overrides `PAPERSPLZ_HOME`; there is no implicit default. `init` is
the exception because the catalog path is its argument:

```sh
pplz init "$HOME/papers/my-catalog" \
  --name "My Papers" \
  --description "Papers and working notes"
```

A catalog contains `catalog.json`, `papers/`, one `record.json` per paper, and
catalog-owned document copies. Move or copy the whole directory to transfer it
to another supported system, then select its new path.

## Add and inspect papers

Add a local file or direct HTTP/HTTPS document URL. A title is required;
authors and tags may repeat:

```sh
pplz add "$HOME/Downloads/paper.pdf" \
  --title "Some Interesting Paper" \
  --author "Alice Smith" --author "Bob Jones" \
  --source "Journal of Interesting Things" \
  --tag topology --tag to-read

pplz add "https://example.org/paper.pdf" --title "A Remote Paper"
```

The catalog owns an independent copy. Imports record size and SHA-256 and
reject duplicate content. URL import downloads the supplied URL directly; it
does not scrape web pages.

```sh
pplz list
pplz list --tag topology --author alice
pplz list --tag topology --sort author --limit 20
pplz list --sort added --reverse
pplz info
pplz show PAPER_ID
pplz edit PAPER_ID --title "Corrected title" --author "Alice Smith"
pplz edit PAPER_ID --clear-authors
pplz open PAPER_ID
pplz path PAPER_ID
open "$(pplz path PAPER_ID)"       # macOS
xdg-open "$(pplz path PAPER_ID)"  # common Unix desktops
```

Paper IDs may be replaced by an unambiguous prefix.

`edit` changes only the supplied metadata fields. Use `--clear-authors` to
remove every author; it cannot be combined with `--author`.

`open` launches the catalog-owned document with `open` on macOS or `xdg-open`
on Linux. On FreeBSD it uses `xdg-open`, falling back to `gio open`. If no
suitable program is installed or it cannot be launched, papersplz reports the
stored path so it can be opened manually. Use `path` for scripts.

Export all catalog-managed metadata as portable JSON (document contents are
not embedded):

```sh
pplz export > catalog-export.json
pplz export --json
```

`list` sorts by title ascending by default. `--sort added` orders oldest first,
and `--sort author` orders by the complete author list with papers that have no
authors last. `--reverse` reverses the selected deterministic order. Equal
primary keys are ordered by title and then paper ID. `--limit` accepts a
positive result count and is applied after filtering and sorting.

## Reviews, comments, and tags

Each paper has at most one review:

```sh
pplz review set PAPER_ID "Review text"
pplz review set PAPER_ID --file review.txt
cat review.txt | pplz review set PAPER_ID -
pplz review show PAPER_ID
EDITOR=vi pplz review edit PAPER_ID
pplz review remove PAPER_ID
```

Comments are separate timestamped working notes. Comment IDs may be exact or
unambiguous prefixes within the paper:

```sh
pplz comment add PAPER_ID "Check the proof of Lemma 4"
pplz comment list PAPER_ID
pplz comment show PAPER_ID COMMENT_ID
EDITOR=vi pplz comment edit PAPER_ID COMMENT_ID
pplz comment remove PAPER_ID COMMENT_ID
```

```sh
pplz tag add PAPER_ID topology to-read
pplz tag list PAPER_ID
pplz tag remove PAPER_ID to-read
pplz tags
```

Tags are normalized to lowercase. `pplz tags` lists catalog-wide usage by count
descending, then tag name ascending. Editing requires `EDITOR`; the program
does not guess an editor.

Reading status uses the conventional tags `to-read`, `reading`, and `read`:

```sh
pplz tag remove PAPER_ID to-read
pplz tag add PAPER_ID reading
pplz list --tag reading
```

These are ordinary, user-managed tags rather than a separate status field.

## Paper relationships

Relationships use a fixed vocabulary and reference papers in the same catalog:

```sh
pplz relation add PAPER_ID cites OTHER_PAPER_ID
pplz relation add PAPER_ID related-to OTHER_PAPER_ID
pplz relation list PAPER_ID
pplz relation list PAPER_ID --json
pplz relation remove PAPER_ID cites OTHER_PAPER_ID
```

Inverse views such as `cited-by` and `superseded-by` are derived automatically.
Supported types are `related-to`, `cites`, `cited-by`, `supersedes`, and
`superseded-by`. `related-to` is symmetric; the other types form the indicated
directional inverse pairs. Only one direction is stored and the inverse view is
derived. Removing a paper removes references to it. Relationship support uses
catalog and paper schema version 2; adding the first relationship safely
upgrades an existing schema-version-1 catalog.

## Search

Search is case-insensitive plain text over titles, authors, sources, tags,
review text, and comment text. It does not search document contents. Multiple
terms use AND semantics and may match different fields:

```sh
pplz search spectral sequence
pplz search serre --tag topology --author alice
pplz search --tag to-read
```

Search performs a linear metadata scan and creates no index.

## JSON output

Structured output is available for `export`, `info`, `list`, `show`, `search`,
`tags`, `review show`, `comment list`, `comment show`, and `tag list`:

```sh
pplz info --json
pplz list --json
pplz show PAPER_ID --json
pplz search topology --json
pplz review show PAPER_ID --json
pplz comment list PAPER_ID --json
pplz comment show PAPER_ID COMMENT_ID --json
pplz tag list PAPER_ID --json
pplz tags --json
```

Diagnostics go to stderr and normal results to stdout. `path` prints only the
document path and a trailing newline.

## Remove papers

Removal deletes the complete catalog-owned paper directory, including its
metadata and document. It never changes the original imported file:

```sh
pplz remove PAPER_ID       # prompts when interactive
pplz remove PAPER_ID --yes # required when non-interactive
```

## Check catalog consistency

`doctor` performs a read-only scan of metadata, documents, hashes, sizes,
duplicates, and unexpected or incomplete paper directories:

```sh
pplz doctor
```

It reports all safely discoverable problems. Its exit status is zero for a
clean catalog and non-zero when problems are found.
