// Package cli parses command-line arguments and dispatches papersplz commands.
package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/catalog"
)

const usage = `Usage: papersplz [--home PATH] COMMAND [ARGUMENTS]

Commands:
  init PATH       create a catalog
  add SOURCE      add a local or direct-URL paper
  remove PAPER    remove a paper
  show PAPER      show a paper summary
  path PAPER      print a paper's stored path
  list            list papers
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
	return run(args, nil, false, stdout, stderr, lookupEnv)
}

// RunWithInput executes the CLI using stdin and detects whether it is a
// terminal for commands requiring interactive confirmation.
func RunWithInput(args []string, stdin io.Reader, stdout, stderr io.Writer, lookupEnv func(string) (string, bool)) int {
	return run(args, stdin, isTerminalInput(stdin), stdout, stderr, lookupEnv)
}

func run(args []string, stdin io.Reader, interactive bool, stdout, stderr io.Writer, lookupEnv func(string) (string, bool)) int {
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

	catalogHome, err := catalog.ResolveHome(*home, lookupEnv)
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: %v\n", err)
		return 1
	}
	switch command {
	case "add":
		return runAdd(catalogHome, commandArgs, stdout, stderr)
	case "remove":
		return runRemove(catalogHome, commandArgs, stdin, interactive, stdout, stderr)
	case "show":
		return runShow(catalogHome, commandArgs, stdout, stderr)
	case "list":
		return runList(catalogHome, commandArgs, stdout, stderr)
	case "path":
		return runPath(catalogHome, commandArgs, stdout, stderr)
	}

	fmt.Fprintf(stderr, "papersplz: %s is not implemented\n", command)
	return 1
}

func runRemove(catalogHome string, args []string, stdin io.Reader, interactive bool, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "papersplz: remove requires PAPER")
		return 2
	}
	selector := args[0]
	flags := flag.NewFlagSet("papersplz remove", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	yes := flags.Bool("yes", false, "remove without prompting")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "papersplz: remove accepts one PAPER")
		return 2
	}
	if !interactive && !*yes {
		fmt.Fprintln(stderr, "papersplz: remove requires --yes when stdin is not a terminal")
		return 1
	}
	if interactive && !*yes {
		paper, err := catalog.LoadPaper(catalogHome, selector)
		if err != nil {
			fmt.Fprintf(stderr, "papersplz: remove: %v\n", err)
			return 1
		}
		fmt.Fprintf(stderr, "Remove %s %q and its catalog-owned files? [y/N] ", paper.ID, paper.Title)
		scanner := bufio.NewScanner(stdin)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(stderr, "papersplz: remove: read confirmation: %v\n", err)
				return 1
			}
			fmt.Fprintln(stdout, "Not removed.")
			return 0
		}
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(stdout, "Not removed.")
			return 0
		}
	}
	paper, err := catalog.RemovePaper(catalogHome, selector)
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: remove: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Removed %s %s\n", paper.ID, paper.Title)
	return 0
}

func isTerminalInput(input io.Reader) bool {
	file, ok := input.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func runShow(catalogHome string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "papersplz: show requires PAPER")
		return 2
	}
	selector := args[0]
	flags := flag.NewFlagSet("papersplz show", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "papersplz: show accepts one PAPER")
		return 2
	}
	paper, err := catalog.LoadPaper(catalogHome, selector)
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: show: %v\n", err)
		return 1
	}
	if *jsonOutput {
		if err := writeJSON(stdout, newShowOutput(paper)); err != nil {
			fmt.Fprintf(stderr, "papersplz: show: write JSON: %v\n", err)
			return 1
		}
	} else {
		writeShow(stdout, paper)
	}
	return 0
}

func runList(catalogHome string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("papersplz list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	tag := flags.String("tag", "", "filter by tag")
	author := flags.String("author", "", "filter by author text")
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "papersplz: list does not accept arguments")
		return 2
	}
	papers, err := catalog.ListPapers(catalogHome, catalog.ListOptions{Tag: *tag, Author: *author})
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: list: %v\n", err)
		return 1
	}
	if *jsonOutput {
		if err := writeJSON(stdout, newListOutput(papers)); err != nil {
			fmt.Fprintf(stderr, "papersplz: list: write JSON: %v\n", err)
			return 1
		}
	} else if err := writeList(stdout, papers); err != nil {
		fmt.Fprintf(stderr, "papersplz: list: write output: %v\n", err)
		return 1
	}
	return 0
}

func runPath(catalogHome string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "papersplz: path requires one PAPER")
		return 2
	}
	documentPath, err := catalog.DocumentPath(catalogHome, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: path: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, documentPath)
	return 0
}

type repeatedStrings []string

func (values *repeatedStrings) String() string { return strings.Join(*values, ",") }

func (values *repeatedStrings) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func runAdd(catalogHome string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "papersplz: add requires SOURCE")
		return 2
	}
	sourcePath := args[0]
	flags := flag.NewFlagSet("papersplz add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	title := flags.String("title", "", "paper title (required)")
	source := flags.String("source", "", "bibliographic source")
	var authors, tags repeatedStrings
	flags.Var(&authors, "author", "author name (repeatable)")
	flags.Var(&tags, "tag", "tag (repeatable)")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "papersplz: add accepts one SOURCE")
		return 2
	}
	if *title == "" {
		fmt.Fprintln(stderr, "papersplz: add requires --title TITLE")
		return 2
	}
	paper, err := catalog.Add(catalogHome, sourcePath, catalog.AddOptions{
		Title:   *title,
		Authors: authors,
		Source:  *source,
		Tags:    tags,
	}, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: add: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Added %s %s\n", paper.ID, paper.Title)
	return 0
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
