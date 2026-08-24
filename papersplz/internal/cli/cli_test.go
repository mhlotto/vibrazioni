package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/catalog"
	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func runForTest(args []string, env map[string]string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	lookup := func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}
	status := Run(args, &stdout, &stderr, lookup)
	return status, stdout.String(), stderr.String()
}

func runInteractiveForTest(args []string, input string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	status := run(args, strings.NewReader(input), true, &stdout, &stderr, func(string) (string, bool) {
		return "", false
	})
	return status, stdout.String(), stderr.String()
}

func TestCatalogCommandRequiresHome(t *testing.T) {
	status, stdout, stderr := runForTest([]string{"list"}, nil)
	if status == 0 {
		t.Fatal("Run() status = 0, want non-zero")
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "--home PATH") || !strings.Contains(stderr, "PAPERSPLZ_HOME") {
		t.Fatalf("stderr = %q, want catalog selection guidance", stderr)
	}
}

func TestCatalogSourcesAllowDispatch(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
	}{
		{name: "home flag", args: []string{"--home", "/flag/catalog", "review"}},
		{name: "environment", args: []string{"review"}, env: map[string]string{"PAPERSPLZ_HOME": "/env/catalog"}},
		{name: "flag overrides environment", args: []string{"--home", "/flag/catalog", "review"}, env: map[string]string{"PAPERSPLZ_HOME": "/env/catalog"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, stdout, stderr := runForTest(tt.args, tt.env)
			if status == 0 {
				t.Fatal("Run() status = 0 while command is a stub")
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if strings.Contains(stderr, "catalog home is required") {
				t.Fatalf("stderr = %q, catalog should have resolved", stderr)
			}
			if !strings.Contains(stderr, "not implemented") {
				t.Fatalf("stderr = %q, want dispatch result", stderr)
			}
		})
	}
}

func TestInitDoesNotRequireCatalogHome(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog")
	status, stdout, stderr := runForTest([]string{"init", path, "--name", "Mathematics", "--description", "Papers"}, nil)
	if status != 0 {
		t.Fatalf("Run() status = %d, want 0; stderr = %q", status, stderr)
	}
	if !strings.Contains(stdout, path) {
		t.Fatalf("stdout = %q, want catalog path", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if strings.Contains(stderr, "catalog home is required") {
		t.Fatalf("stderr = %q, init must bypass catalog selection", stderr)
	}
	if _, err := os.Stat(filepath.Join(path, store.CatalogFilename)); err != nil {
		t.Fatalf("catalog metadata was not created: %v", err)
	}
}

func TestAddLocalCommand(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), "catalog")
	if status, _, stderr := runForTest([]string{"init", catalogPath, "--name", "Test"}, nil); status != 0 {
		t.Fatalf("init failed: %s", stderr)
	}
	sourcePath := filepath.Join(t.TempDir(), "paper.pdf")
	if err := os.WriteFile(sourcePath, []byte("paper contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, stdout, stderr := runForTest([]string{
		"--home", catalogPath, "add", sourcePath,
		"--title", "Paper", "--author", "Alice", "--author", "Bob",
		"--source", "Journal", "--tag", "Math", "--tag", "To-Read",
	}, nil)
	if status != 0 {
		t.Fatalf("Run() status = %d, stderr = %q", status, stderr)
	}
	if !strings.HasPrefix(stdout, "Added ") || stderr != "" {
		t.Fatalf("stdout = %q, stderr = %q", stdout, stderr)
	}
	entries, err := os.ReadDir(filepath.Join(catalogPath, "papers"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("paper directories = %v, error = %v", entries, err)
	}
	paper, err := store.ReadPaper(filepath.Join(catalogPath, "papers", entries[0].Name(), store.RecordFilename))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(paper.Authors, ",") != "Alice,Bob" || strings.Join(paper.Tags, ",") != "math,to-read" {
		t.Fatalf("paper metadata = %#v", paper)
	}
}

func TestShowListAndPathCommands(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), "catalog")
	if err := catalog.Initialize(catalogPath, "Test", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 8, 24, 15, 31, 0, 0, time.UTC)
	papers := []model.Paper{
		cliFixturePaper("aaaa000000000000", "Zoo", []string{"Carol"}, []string{"physics"}, created),
		cliFixturePaper("bbbb000000000000", "alpha", []string{"Alice Smith"}, []string{"math"}, created),
		cliFixturePaper("cccc000000000000", "Beta", []string{"Bob", "Alice Cooper"}, []string{"math", "topology"}, created),
	}
	papers[1].Source = "Journal"
	papers[1].SourceURL = "https://example.org/alpha.pdf"
	papers[1].Review = &model.Review{Text: "Reviewed", CreatedAt: created, UpdatedAt: created}
	papers[1].Comments = []model.Comment{{ID: "dddd0000", Text: "Note", CreatedAt: created, UpdatedAt: created}}
	for _, paper := range papers {
		writeCLIFixturePaper(t, catalogPath, paper)
	}

	t.Run("show human with prefix", func(t *testing.T) {
		status, stdout, stderr := runForTest([]string{"--home", catalogPath, "show", "bbbb"}, nil)
		if status != 0 || stderr != "" {
			t.Fatalf("status = %d, stderr = %q", status, stderr)
		}
		for _, expected := range []string{
			"ID: bbbb000000000000", "Title: alpha", "Authors: Alice Smith",
			"Source: Journal", "Source URL: https://example.org/alpha.pdf",
			"Tags: math", "Added: 2026-08-24T15:31:00Z", "Updated: 2026-08-24T15:31:00Z",
			"Stored filename: paper.pdf", "Original filename: alpha.pdf",
			"File size: 8 bytes", "Review: present", "Comments: 1",
		} {
			if !strings.Contains(stdout, expected) {
				t.Fatalf("show output missing %q:\n%s", expected, stdout)
			}
		}
	})

	t.Run("show json structure", func(t *testing.T) {
		status, stdout, stderr := runForTest([]string{"--home", catalogPath, "show", "bbbb", "--json"}, nil)
		if status != 0 || stderr != "" {
			t.Fatalf("status = %d, stderr = %q", status, stderr)
		}
		var output ShowOutput
		if err := json.Unmarshal([]byte(stdout), &output); err != nil {
			t.Fatal(err)
		}
		if output.ID != papers[1].ID || output.ReviewStatus != "present" || output.CommentCount != 1 {
			t.Fatalf("show JSON = %#v", output)
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(stdout), &object); err != nil {
			t.Fatal(err)
		}
		if len(object) != 13 {
			t.Fatalf("show JSON keys = %v", object)
		}
	})

	t.Run("list human deterministic order", func(t *testing.T) {
		status, stdout, stderr := runForTest([]string{"--home", catalogPath, "list"}, nil)
		if status != 0 || stderr != "" {
			t.Fatalf("status = %d, stderr = %q", status, stderr)
		}
		alpha, beta, zoo := strings.Index(stdout, "alpha"), strings.Index(stdout, "Beta"), strings.Index(stdout, "Zoo")
		if alpha < 0 || !(alpha < beta && beta < zoo) {
			t.Fatalf("list is not in title order:\n%s", stdout)
		}
		if !strings.Contains(stdout, "bbbb0000") || strings.Contains(stdout, papers[1].ID) {
			t.Fatalf("list does not use concise IDs:\n%s", stdout)
		}
	})

	t.Run("list json and filters", func(t *testing.T) {
		status, stdout, stderr := runForTest([]string{"--home", catalogPath, "list", "--tag", "MATH", "--author", "alice", "--json"}, nil)
		if status != 0 || stderr != "" {
			t.Fatalf("status = %d, stderr = %q", status, stderr)
		}
		var output []ListOutput
		if err := json.Unmarshal([]byte(stdout), &output); err != nil {
			t.Fatal(err)
		}
		want := []ListOutput{
			{ID: papers[1].ID, Title: "alpha", Authors: []string{"Alice Smith"}},
			{ID: papers[2].ID, Title: "Beta", Authors: []string{"Bob", "Alice Cooper"}},
		}
		if !reflect.DeepEqual(output, want) {
			t.Fatalf("list JSON = %#v, want %#v", output, want)
		}
	})

	t.Run("path only prints document path", func(t *testing.T) {
		status, stdout, stderr := runForTest([]string{"--home", catalogPath, "path", "bbbb"}, nil)
		want := filepath.Join(catalogPath, catalog.PapersDirectory, papers[1].ID, papers[1].File.Name) + "\n"
		if status != 0 || stderr != "" || stdout != want {
			t.Fatalf("status = %d, stdout = %q, stderr = %q, want %q", status, stdout, stderr, want)
		}
	})
}

func TestRemoveCommandConfirmation(t *testing.T) {
	t.Run("non-interactive requires yes", func(t *testing.T) {
		catalogPath, paper := removeCLIFixture(t)
		status, stdout, stderr := runForTest([]string{"--home", catalogPath, "remove", paper.ID[:8]}, nil)
		if status == 0 || stdout != "" || !strings.Contains(stderr, "requires --yes") {
			t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
		}
		if _, err := os.Stat(filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID)); err != nil {
			t.Fatalf("paper removed without confirmation: %v", err)
		}
	})

	t.Run("non-interactive yes removes", func(t *testing.T) {
		catalogPath, paper := removeCLIFixture(t)
		status, stdout, stderr := runForTest([]string{"--home", catalogPath, "remove", paper.ID[:8], "--yes"}, nil)
		if status != 0 || stderr != "" || !strings.Contains(stdout, "Removed "+paper.ID) {
			t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
		}
		if _, err := os.Stat(filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("paper directory still exists: %v", err)
		}
	})

	t.Run("interactive decline is safe", func(t *testing.T) {
		catalogPath, paper := removeCLIFixture(t)
		status, stdout, stderr := runInteractiveForTest([]string{"--home", catalogPath, "remove", paper.ID[:8]}, "no\n")
		if status != 0 || stdout != "Not removed.\n" || !strings.Contains(stderr, "[y/N]") {
			t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
		}
		if _, err := os.Stat(filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID)); err != nil {
			t.Fatalf("paper removed after decline: %v", err)
		}
	})

	t.Run("interactive confirmation removes", func(t *testing.T) {
		catalogPath, paper := removeCLIFixture(t)
		status, stdout, stderr := runInteractiveForTest([]string{"--home", catalogPath, "remove", paper.ID[:8]}, "YES\n")
		if status != 0 || !strings.Contains(stdout, "Removed "+paper.ID) || !strings.Contains(stderr, paper.Title) {
			t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
		}
		if _, err := os.Stat(filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("paper directory still exists: %v", err)
		}
	})
}

func TestTagCommands(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), "catalog")
	if err := catalog.Initialize(catalogPath, "Test", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	paper := cliFixturePaper("abcd000000000000", "Tagged Paper", nil, []string{"math"}, time.Now().UTC())
	writeCLIFixturePaper(t, catalogPath, paper)

	status, stdout, stderr := runForTest([]string{
		"--home", catalogPath, "tag", "add", "abcd", " Topology ", "MATH", "qft+notes",
	}, nil)
	if status != 0 || stderr != "" || stdout != "math\ntopology\nqft+notes\n" {
		t.Fatalf("tag add: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "tag", "list", "abcd"}, nil)
	if status != 0 || stderr != "" || stdout != "math\ntopology\nqft+notes\n" {
		t.Fatalf("tag list: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "tag", "list", "abcd", "--json"}, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("tag list JSON: status = %d, stderr = %q", status, stderr)
	}
	var output TagListOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatal(err)
	}
	want := TagListOutput{PaperID: paper.ID, Tags: []string{"math", "topology", "qft+notes"}}
	if !reflect.DeepEqual(output, want) {
		t.Fatalf("tag list JSON = %#v, want %#v", output, want)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &object); err != nil || len(object) != 2 {
		t.Fatalf("tag list JSON structure = %v, error = %v", object, err)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "tag", "remove", "abcd", "MATH", "missing"}, nil)
	if status != 0 || stderr != "" || stdout != "topology\nqft+notes\n" {
		t.Fatalf("tag remove: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}

	before, err := os.ReadFile(filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID, store.RecordFilename))
	if err != nil {
		t.Fatal(err)
	}
	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "tag", "add", "abcd", "bad tag"}, nil)
	if status == 0 || stdout != "" || !strings.Contains(stderr, "invalid tag") {
		t.Fatalf("invalid tag: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	after, err := os.ReadFile(filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID, store.RecordFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("invalid CLI tag changed record.json")
	}
}

func removeCLIFixture(t *testing.T) (string, model.Paper) {
	t.Helper()
	catalogPath := filepath.Join(t.TempDir(), "catalog")
	if err := catalog.Initialize(catalogPath, "Test", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	paper := cliFixturePaper("abcd000000000000", "Disposable Paper", []string{"Alice"}, []string{"test"}, time.Now().UTC())
	writeCLIFixturePaper(t, catalogPath, paper)
	return catalogPath, paper
}

func cliFixturePaper(id, title string, authors, tags []string, timestamp time.Time) model.Paper {
	return model.Paper{
		SchemaVersion: model.SchemaVersion,
		ID:            id,
		Title:         title,
		Authors:       authors,
		AddedAt:       timestamp,
		UpdatedAt:     timestamp,
		File: model.File{
			Name:         "paper.pdf",
			OriginalName: title + ".pdf",
			Size:         8,
			SHA256:       strings.Repeat("a", 64),
		},
		Tags:     tags,
		Comments: []model.Comment{},
	}
}

func writeCLIFixturePaper(t *testing.T, catalogPath string, paper model.Paper) {
	t.Helper()
	directory := filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID)
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, paper.File.Name), []byte("document"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.WritePaper(filepath.Join(directory, store.RecordFilename), paper); err != nil {
		t.Fatal(err)
	}
}

func TestBasicErrorsUseStderr(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "no command", want: "Usage:"},
		{name: "unknown command", args: []string{"nope"}, want: "unknown command"},
		{name: "missing home value", args: []string{"--home"}, want: "flag needs an argument"},
		{name: "init missing path", args: []string{"init"}, want: "init requires PATH"},
		{name: "init missing name", args: []string{"init", t.TempDir()}, want: "init requires --name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, stdout, stderr := runForTest(tt.args, nil)
			if status == 0 {
				t.Fatal("Run() status = 0, want non-zero")
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, tt.want) {
				t.Fatalf("stderr = %q, want substring %q", stderr, tt.want)
			}
		})
	}
}

func TestHelpUsesStdout(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}} {
		status, stdout, stderr := runForTest(args, nil)
		if status != 0 {
			t.Fatalf("Run(%q) status = %d, want 0", args, status)
		}
		if !strings.Contains(stdout, "Usage:") {
			t.Fatalf("Run(%q) stdout = %q, want usage", args, stdout)
		}
		if stderr != "" {
			t.Fatalf("Run(%q) stderr = %q, want empty", args, stderr)
		}
	}
}
