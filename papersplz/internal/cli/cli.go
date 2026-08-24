// Package cli parses command-line arguments and dispatches papersplz commands.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/mhlotto/vibrazioni/papersplz/internal/catalog"
)

const usage = `Usage: papersplz [--home PATH] COMMAND [ARGUMENTS]

Commands:
  init PATH       create a catalog (not yet implemented)
  add             add a paper (not yet implemented)
  remove          remove a paper (not yet implemented)
  show            show a paper (not yet implemented)
  path            print a paper's stored path (not yet implemented)
  list            list papers (not yet implemented)
  review          manage reviews (not yet implemented)
  comment         manage comments (not yet implemented)
  tag             manage tags (not yet implemented)
  search          search papers (not yet implemented)
  doctor          check catalog consistency (not yet implemented)
`

var catalogCommands = map[string]struct{}{
	"add": {}, "remove": {}, "show": {}, "path": {}, "list": {},
	"review": {}, "comment": {}, "tag": {}, "search": {}, "doctor": {},
}

// Run executes the command-line interface and returns a process exit status.
func Run(args []string, stdout, stderr io.Writer, lookupEnv func(string) (string, bool)) int {
	flags := flag.NewFlagSet("papersplz", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	home := flags.String("home", "", "catalog path (overrides PAPERSPLZ_HOME)")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return 0
		}
		return 2
	}
	remaining := flags.Args()
	if len(remaining) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	command := remaining[0]
	commandArgs := remaining[1:]
	switch command {
	case "help":
		if len(commandArgs) != 0 {
			fmt.Fprintln(stderr, "papersplz: help does not accept arguments")
			return 2
		}
		fmt.Fprint(stdout, usage)
		return 0
	case "init":
		if len(commandArgs) == 0 {
			fmt.Fprintln(stderr, "papersplz: init requires PATH")
			return 2
		}
		fmt.Fprintln(stderr, "papersplz: init is not implemented")
		return 1
	}

	if _, ok := catalogCommands[command]; !ok {
		fmt.Fprintf(stderr, "papersplz: unknown command %q\n", command)
		return 2
	}

	if _, err := catalog.ResolveHome(*home, lookupEnv); err != nil {
		fmt.Fprintf(stderr, "papersplz: %v\n", err)
		return 1
	}

	fmt.Fprintf(stderr, "papersplz: %s is not implemented\n", command)
	return 1
}
