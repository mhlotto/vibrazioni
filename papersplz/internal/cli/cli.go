// Package cli parses command-line arguments and dispatches papersplz commands.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/catalog"
)

const usage = `Usage: papersplz [--home PATH] COMMAND [ARGUMENTS]

Commands:
  init PATH       create a catalog
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
		return runInit(commandArgs, stdout, stderr)
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

func runInit(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "papersplz: init requires PATH")
		return 2
	}
	path := args[0]
	flags := flag.NewFlagSet("papersplz init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	name := flags.String("name", "", "catalog name (required)")
	description := flags.String("description", "", "catalog description")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "papersplz: init accepts one PATH")
		return 2
	}
	if *name == "" {
		fmt.Fprintln(stderr, "papersplz: init requires --name NAME")
		return 2
	}
	if err := catalog.Initialize(path, *name, *description, time.Now().UTC()); err != nil {
		fmt.Fprintf(stderr, "papersplz: init: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Initialized catalog at %s\n", path)
	return 0
}
