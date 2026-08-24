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
	listHelp = `Usage: papersplz list [OPTIONS]

List papers in deterministic title order.

Options:
  --tag TAG        filter by exact tag
  --author TEXT    filter by author text
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
)

var topLevelHelp = map[string]string{
	"init": initHelp, "add": addHelp, "edit": editHelp, "remove": removeHelp,
	"show": showHelp, "path": pathHelp, "list": listHelp,
	"info":   infoHelp,
	"search": searchHelp, "doctor": doctorHelp,
	"review": reviewHelp, "comment": commentHelp, "tag": tagHelp,
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
