package cli

import "strings"

const (
	initHelp = `Usage: papersplz init PATH --name NAME [--description DESCRIPTION]

Create a new self-contained catalog at PATH.
`
	addHelp = `Usage: papersplz add SOURCE --title TITLE [OPTIONS]

Add a local file or direct HTTP/HTTPS document URL.

Options:
  --title TITLE    paper title (required)
  --author NAME    author name (repeatable)
  --source TEXT    bibliographic source
  --tag TAG        tag (repeatable)
`
	editHelp = `Usage: papersplz edit PAPER [OPTIONS]

Change descriptive paper metadata without replacing the stored document.

Options:
  --title TITLE       replace paper title
  --author NAME       replace authors (repeatable)
  --clear-authors     remove all authors (cannot combine with --author)
  --source TEXT       replace bibliographic source
  --source-url URL    replace source URL
`
	removeHelp = `Usage: papersplz remove PAPER [--yes]

Remove a catalog-owned paper and its metadata.

Options:
  --yes            remove without prompting
`
	showHelp = `Usage: papersplz show PAPER [--json]

Show a paper metadata summary.
`
	pathHelp = `Usage: papersplz path PAPER

Print only the stored document path.
`
	openHelp = `Usage: papersplz open PAPER

Open the stored document using the platform's local viewer.
`
	listHelp = `Usage: papersplz list [OPTIONS]

List papers in deterministic order. The default is title ascending.

Options:
  --tag TAG        filter by exact tag
  --author TEXT    filter by author text
  --sort FIELD     sort by title, added, or author (default title)
  --reverse        reverse the selected ordering
  --limit N        return at most N papers (N must be positive)
  --json           print JSON
`
	infoHelp = `Usage: papersplz info [--json]

Show catalog metadata, counts, path, and last-added date.
`
	searchHelp = `Usage: papersplz search TERM... [OPTIONS]

Search title, authors, source, tags, review text, and comment text.
Terms use case-insensitive plain-text AND matching.

Options:
  --tag TAG        filter by exact tag
  --author TEXT    filter by author text
  --json           print JSON
`
	doctorHelp = `Usage: papersplz doctor

Check catalog consistency without modifying it.
`
	exportHelp = `Usage: papersplz export [--json]

Export catalog metadata and all paper records as JSON. JSON is the default
format; stored document contents are not included.
`
	reviewHelp = `Usage: papersplz review COMMAND [ARGUMENTS]

Commands:
  show PAPER [--json]
  set PAPER TEXT
  set PAPER --file FILE
  set PAPER -
  edit PAPER
  remove PAPER
`
	commentHelp = `Usage: papersplz comment COMMAND [ARGUMENTS]

Commands:
  add PAPER TEXT
  list PAPER [--json]
  show PAPER COMMENT [--json]
  edit PAPER COMMENT
  remove PAPER COMMENT
`
	tagHelp = `Usage: papersplz tag COMMAND [ARGUMENTS]

Commands:
  add PAPER TAG...
  remove PAPER TAG...
  list PAPER [--json]
`
	tagsHelp = `Usage: papersplz tags [--json]

List all catalog tags with paper usage counts. Results are ordered by count
descending, then tag name ascending.
`
	relationHelp = `Usage: papersplz relation COMMAND [ARGUMENTS]

Commands:
  add PAPER TYPE OTHER
  list PAPER [--json]
  remove PAPER TYPE OTHER

Types: related-to, cites, cited-by, supersedes, superseded-by
`
)

var topLevelHelp = map[string]string{
	"init": initHelp, "add": addHelp, "edit": editHelp, "remove": removeHelp,
	"show": showHelp, "path": pathHelp, "open": openHelp, "list": listHelp,
	"info":   infoHelp,
	"search": searchHelp, "export": exportHelp, "doctor": doctorHelp,
	"review": reviewHelp, "comment": commentHelp, "tag": tagHelp,
	"tags": tagsHelp, "relation": relationHelp,
}

var nestedHelp = map[string]map[string]string{
	"review": {
		"show":   "Usage: papersplz review show PAPER [--json]\n",
		"set":    "Usage: papersplz review set PAPER (TEXT | --file FILE | -)\n",
		"edit":   "Usage: papersplz review edit PAPER\n\nOpen the review using $EDITOR.\n",
		"remove": "Usage: papersplz review remove PAPER\n",
	},
	"comment": {
		"add":    "Usage: papersplz comment add PAPER TEXT\n",
		"list":   "Usage: papersplz comment list PAPER [--json]\n",
		"show":   "Usage: papersplz comment show PAPER COMMENT [--json]\n",
		"edit":   "Usage: papersplz comment edit PAPER COMMENT\n\nOpen the comment using $EDITOR.\n",
		"remove": "Usage: papersplz comment remove PAPER COMMENT\n",
	},
	"tag": {
		"add":    "Usage: papersplz tag add PAPER TAG...\n",
		"remove": "Usage: papersplz tag remove PAPER TAG...\n",
		"list":   "Usage: papersplz tag list PAPER [--json]\n",
	},
	"relation": {
		"add":    "Usage: papersplz relation add PAPER TYPE OTHER\n",
		"list":   "Usage: papersplz relation list PAPER [--json]\n",
		"remove": "Usage: papersplz relation remove PAPER TYPE OTHER\n",
	},
}

func commandHelp(command string, args []string) (string, bool) {
	groupHelp, known := topLevelHelp[command]
	if !known || len(args) == 0 {
		return "", false
	}
	if isHelpFlag(args[0]) {
		return groupHelp, true
	}
	if commands, ok := nestedHelp[command]; ok && len(args) > 1 && isHelpFlag(args[1]) {
		help, ok := commands[strings.ToLower(args[0])]
		return help, ok
	}
	return "", false
}

func isHelpFlag(argument string) bool {
	return argument == "--help" || argument == "-h"
}
