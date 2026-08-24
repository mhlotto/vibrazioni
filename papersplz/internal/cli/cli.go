// Package cli parses command-line arguments and dispatches papersplz commands.
package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/catalog"
	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

const usage = `Usage: papersplz [--home PATH] COMMAND [ARGUMENTS]

Commands:
  init PATH       create a catalog
  add SOURCE      add a local or direct-URL paper
  remove PAPER    remove a paper
  show PAPER      show a paper summary
  path PAPER      print a paper's stored path
  list            list papers
  review          show, set, edit, or remove a review
  comment         add, list, show, edit, or remove comments
  tag             add, remove, or list paper tags
  search          search paper metadata
  doctor          check catalog consistency
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
	case "tag":
		return runTag(catalogHome, commandArgs, stdout, stderr)
	case "review":
		return runReview(catalogHome, commandArgs, stdin, stdout, stderr, lookupEnv)
	case "comment":
		return runComment(catalogHome, commandArgs, stdin, stdout, stderr, lookupEnv)
	case "search":
		return runSearch(catalogHome, commandArgs, stdout, stderr)
	case "doctor":
		return runDoctor(catalogHome, commandArgs, stdout, stderr)
	}

	fmt.Fprintf(stderr, "papersplz: %s is not implemented\n", command)
	return 1
}

func runDoctor(catalogHome string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "papersplz: doctor does not accept arguments")
		return 2
	}
	problems := catalog.Doctor(catalogHome)
	if len(problems) == 0 {
		fmt.Fprintln(stdout, "Catalog is healthy.")
		return 0
	}
	for _, problem := range problems {
		fmt.Fprintf(stdout, "problem: %s\n", problem)
	}
	fmt.Fprintf(stdout, "%d problem(s) found.\n", len(problems))
	return 1
}

func runSearch(catalogHome string, args []string, stdout, stderr io.Writer) int {
	options, jsonOutput, err := parseSearchArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: search: %v\n", err)
		return 2
	}
	papers, err := catalog.SearchPapers(catalogHome, options)
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: search: %v\n", err)
		return 1
	}
	if jsonOutput {
		if err := writeJSON(stdout, newSearchOutput(papers)); err != nil {
			fmt.Fprintf(stderr, "papersplz: search: write JSON: %v\n", err)
			return 1
		}
	} else if err := writeList(stdout, papers); err != nil {
		fmt.Fprintf(stderr, "papersplz: search: write output: %v\n", err)
		return 1
	}
	return 0
}

func parseSearchArgs(args []string) (catalog.SearchOptions, bool, error) {
	var options catalog.SearchOptions
	jsonOutput := false
	flagsEnabled := true
	for i := 0; i < len(args); i++ {
		argument := args[i]
		if flagsEnabled && argument == "--" {
			flagsEnabled = false
			continue
		}
		if !flagsEnabled || !strings.HasPrefix(argument, "-") || argument == "-" {
			options.Terms = append(options.Terms, argument)
			continue
		}
		switch {
		case argument == "--json":
			jsonOutput = true
		case argument == "--tag" || argument == "--author":
			if i+1 >= len(args) {
				return catalog.SearchOptions{}, false, fmt.Errorf("%s requires a value", argument)
			}
			i++
			if argument == "--tag" {
				options.Tag = args[i]
			} else {
				options.Author = args[i]
			}
		case strings.HasPrefix(argument, "--tag="):
			options.Tag = strings.TrimPrefix(argument, "--tag=")
		case strings.HasPrefix(argument, "--author="):
			options.Author = strings.TrimPrefix(argument, "--author=")
		default:
			return catalog.SearchOptions{}, false, fmt.Errorf("unknown option %q", argument)
		}
	}
	return options, jsonOutput, nil
}

func runComment(catalogHome string, args []string, stdin io.Reader, stdout, stderr io.Writer, lookupEnv func(string) (string, bool)) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "papersplz: comment requires add, list, show, edit, or remove")
		return 2
	}
	switch args[0] {
	case "add":
		if len(args) != 3 {
			fmt.Fprintln(stderr, "papersplz: comment add requires PAPER and TEXT")
			return 2
		}
		paper, comment, err := catalog.AddComment(catalogHome, args[1], args[2], time.Now().UTC())
		if err != nil {
			fmt.Fprintf(stderr, "papersplz: comment add: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Added comment %s to %s.\n", comment.ID, paper.ID)
		return 0
	case "list":
		return runCommentList(catalogHome, args[1:], stdout, stderr)
	case "show":
		return runCommentShow(catalogHome, args[1:], stdout, stderr)
	case "edit":
		return runCommentEdit(catalogHome, args[1:], stdin, stdout, stderr, lookupEnv)
	case "remove":
		if len(args) != 3 {
			fmt.Fprintln(stderr, "papersplz: comment remove requires PAPER and COMMENT")
			return 2
		}
		paper, comment, err := catalog.RemoveComment(catalogHome, args[1], args[2], time.Now().UTC())
		if err != nil {
			fmt.Fprintf(stderr, "papersplz: comment remove: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Removed comment %s from %s.\n", comment.ID, paper.ID)
		return 0
	default:
		fmt.Fprintf(stderr, "papersplz: unknown comment command %q\n", args[0])
		return 2
	}
}

func runCommentList(catalogHome string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "papersplz: comment list requires PAPER")
		return 2
	}
	selector := args[0]
	flags := flag.NewFlagSet("papersplz comment list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "papersplz: comment list accepts one PAPER")
		return 2
	}
	_, comments, err := catalog.ListComments(catalogHome, selector)
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: comment list: %v\n", err)
		return 1
	}
	if *jsonOutput {
		if err := writeJSON(stdout, newCommentListOutput(comments)); err != nil {
			fmt.Fprintf(stderr, "papersplz: comment list: write JSON: %v\n", err)
			return 1
		}
	} else {
		writeComments(stdout, comments)
	}
	return 0
}

func runCommentShow(catalogHome string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "papersplz: comment show requires PAPER and COMMENT")
		return 2
	}
	paperSelector, commentSelector := args[0], args[1]
	flags := flag.NewFlagSet("papersplz comment show", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args[2:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "papersplz: comment show accepts PAPER and COMMENT")
		return 2
	}
	_, comment, err := catalog.ShowComment(catalogHome, paperSelector, commentSelector)
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: comment show: %v\n", err)
		return 1
	}
	if *jsonOutput {
		if err := writeJSON(stdout, newCommentOutput(comment)); err != nil {
			fmt.Fprintf(stderr, "papersplz: comment show: write JSON: %v\n", err)
			return 1
		}
	} else {
		writeComment(stdout, comment)
	}
	return 0
}

func runCommentEdit(catalogHome string, args []string, stdin io.Reader, stdout, stderr io.Writer, lookupEnv func(string) (string, bool)) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "papersplz: comment edit requires PAPER and COMMENT")
		return 2
	}
	editor, ok := lookupEnv("EDITOR")
	if !ok || strings.TrimSpace(editor) == "" {
		fmt.Fprintln(stderr, "papersplz: comment edit: EDITOR is not set")
		return 1
	}
	paper, comment, err := catalog.ShowComment(catalogHome, args[0], args[1])
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: comment edit: %v\n", err)
		return 1
	}
	text, err := editText(comment.Text, "papersplz-comment-*.txt", editor, stdin, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: comment edit: %v\n", err)
		return 1
	}
	_, comment, err = catalog.EditComment(catalogHome, paper.ID, comment.ID, text, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: comment edit: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Updated comment %s.\n", comment.ID)
	return 0
}

func editText(initial, pattern, editor string, stdin io.Reader, stdout, stderr io.Writer) (string, error) {
	temporary, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	_, err = io.WriteString(temporary, initial)
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("prepare temporary file: %w", err)
	}
	command := exec.Command("sh", "-c", `exec $EDITOR "$1"`, "papersplz-editor", temporaryPath)
	command.Env = environmentWith(os.Environ(), "EDITOR", editor)
	command.Stdin, command.Stdout, command.Stderr = stdin, stdout, stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}
	text, err := os.ReadFile(filepath.Clean(temporaryPath))
	if err != nil {
		return "", fmt.Errorf("read temporary file: %w", err)
	}
	return string(text), nil
}

func runReview(catalogHome string, args []string, stdin io.Reader, stdout, stderr io.Writer, lookupEnv func(string) (string, bool)) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "papersplz: review requires show, set, edit, or remove")
		return 2
	}
	switch args[0] {
	case "show":
		return runReviewShow(catalogHome, args[1:], stdout, stderr)
	case "set":
		return runReviewSet(catalogHome, args[1:], stdin, stdout, stderr)
	case "edit":
		return runReviewEdit(catalogHome, args[1:], stdin, stdout, stderr, lookupEnv)
	case "remove":
		return runReviewRemove(catalogHome, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "papersplz: unknown review command %q\n", args[0])
		return 2
	}
}

func runReviewShow(catalogHome string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "papersplz: review show requires PAPER")
		return 2
	}
	selector := args[0]
	flags := flag.NewFlagSet("papersplz review show", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "papersplz: review show accepts one PAPER")
		return 2
	}
	paper, err := catalog.ShowReview(catalogHome, selector)
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: review show: %v\n", err)
		return 1
	}
	if *jsonOutput {
		output := ReviewOutput{
			Text:      paper.Review.Text,
			CreatedAt: paper.Review.CreatedAt,
			UpdatedAt: paper.Review.UpdatedAt,
		}
		if err := writeJSON(stdout, output); err != nil {
			fmt.Fprintf(stderr, "papersplz: review show: write JSON: %v\n", err)
			return 1
		}
	} else {
		writeReview(stdout, *paper.Review)
	}
	return 0
}

func runReviewSet(catalogHome string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "papersplz: review set requires PAPER")
		return 2
	}
	selector := args[0]
	flags := flag.NewFlagSet("papersplz review set", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	filePath := flags.String("file", "", "read review text from a file")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	positional := flags.Args()
	if *filePath != "" && len(positional) != 0 {
		fmt.Fprintln(stderr, "papersplz: review set accepts either TEXT, -, or --file FILE")
		return 2
	}
	if *filePath == "" && len(positional) != 1 {
		fmt.Fprintln(stderr, "papersplz: review set requires TEXT, -, or --file FILE")
		return 2
	}
	var (
		text []byte
		err  error
	)
	if *filePath != "" {
		text, err = os.ReadFile(*filePath)
		if err != nil {
			err = fmt.Errorf("read review file: %w", err)
		}
	} else if positional[0] == "-" {
		if stdin == nil {
			err = errors.New("stdin is unavailable")
		} else {
			text, err = io.ReadAll(stdin)
			if err != nil {
				err = fmt.Errorf("read review from stdin: %w", err)
			}
		}
	} else {
		text = []byte(positional[0])
	}
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: review set: %v\n", err)
		return 1
	}
	paper, err := catalog.SetReview(catalogHome, selector, string(text), time.Now().UTC())
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: review set: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Review set for %s.\n", paper.ID)
	return 0
}

func runReviewEdit(catalogHome string, args []string, stdin io.Reader, stdout, stderr io.Writer, lookupEnv func(string) (string, bool)) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "papersplz: review edit requires one PAPER")
		return 2
	}
	editor, ok := lookupEnv("EDITOR")
	if !ok || strings.TrimSpace(editor) == "" {
		fmt.Fprintln(stderr, "papersplz: review edit: EDITOR is not set")
		return 1
	}
	paper, err := catalog.LoadPaper(catalogHome, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: review edit: %v\n", err)
		return 1
	}
	temporary, err := os.CreateTemp("", "papersplz-review-*.txt")
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: review edit: create temporary file: %v\n", err)
		return 1
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if paper.Review != nil {
		_, err = io.WriteString(temporary, paper.Review.Text)
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: review edit: prepare temporary file: %v\n", err)
		return 1
	}
	command := exec.Command("sh", "-c", `exec $EDITOR "$1"`, "papersplz-editor", temporaryPath)
	command.Env = environmentWith(os.Environ(), "EDITOR", editor)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		fmt.Fprintf(stderr, "papersplz: review edit: editor failed: %v\n", err)
		return 1
	}
	text, err := os.ReadFile(filepath.Clean(temporaryPath))
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: review edit: read temporary file: %v\n", err)
		return 1
	}
	paper, err = catalog.SetReview(catalogHome, paper.ID, string(text), time.Now().UTC())
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: review edit: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Review set for %s.\n", paper.ID)
	return 0
}

func environmentWith(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, prefix) {
			result = append(result, entry)
		}
	}
	return append(result, prefix+value)
}

func runReviewRemove(catalogHome string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "papersplz: review remove requires one PAPER")
		return 2
	}
	paper, err := catalog.RemoveReview(catalogHome, args[0], time.Now().UTC())
	if err != nil {
		fmt.Fprintf(stderr, "papersplz: review remove: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Review removed from %s.\n", paper.ID)
	return 0
}

func runTag(catalogHome string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "papersplz: tag requires add, remove, or list")
		return 2
	}
	switch args[0] {
	case "add", "remove":
		if len(args) < 3 {
			fmt.Fprintf(stderr, "papersplz: tag %s requires PAPER and at least one TAG\n", args[0])
			return 2
		}
		var (
			paper model.Paper
			err   error
		)
		if args[0] == "add" {
			paper, err = catalog.AddTags(catalogHome, args[1], args[2:], time.Now().UTC())
		} else {
			paper, err = catalog.RemoveTags(catalogHome, args[1], args[2:], time.Now().UTC())
		}
		if err != nil {
			fmt.Fprintf(stderr, "papersplz: tag %s: %v\n", args[0], err)
			return 1
		}
		writeTags(stdout, paper.Tags)
		return 0
	case "list":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "papersplz: tag list requires PAPER")
			return 2
		}
		selector := args[1]
		flags := flag.NewFlagSet("papersplz tag list", flag.ContinueOnError)
		flags.SetOutput(stderr)
		flags.Usage = func() {}
		jsonOutput := flags.Bool("json", false, "print JSON")
		if err := flags.Parse(args[2:]); err != nil {
			return 2
		}
		if flags.NArg() != 0 {
			fmt.Fprintln(stderr, "papersplz: tag list accepts one PAPER")
			return 2
		}
		paper, err := catalog.ListTags(catalogHome, selector)
		if err != nil {
			fmt.Fprintf(stderr, "papersplz: tag list: %v\n", err)
			return 1
		}
		if *jsonOutput {
			output := TagListOutput{PaperID: paper.ID, Tags: nonNilStrings(paper.Tags)}
			if err := writeJSON(stdout, output); err != nil {
				fmt.Fprintf(stderr, "papersplz: tag list: write JSON: %v\n", err)
				return 1
			}
		} else {
			writeTags(stdout, paper.Tags)
		}
		return 0
	default:
		fmt.Fprintf(stderr, "papersplz: unknown tag command %q\n", args[0])
		return 2
	}
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
