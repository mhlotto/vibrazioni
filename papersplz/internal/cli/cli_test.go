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

func runWithInputForTest(args []string, input string, env map[string]string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	status := run(args, strings.NewReader(input), false, &stdout, &stderr, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
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
		{name: "home flag", args: []string{"--home", "/flag/catalog", "doctor"}},
		{name: "environment", args: []string{"doctor"}, env: map[string]string{"PAPERSPLZ_HOME": "/env/catalog"}},
		{name: "flag overrides environment", args: []string{"--home", "/flag/catalog", "doctor"}, env: map[string]string{"PAPERSPLZ_HOME": "/env/catalog"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, stdout, stderr := runForTest(tt.args, tt.env)
			if status == 0 {
				t.Fatal("Run() status = 0 for missing catalog")
			}
			if strings.Contains(stderr, "catalog home is required") {
				t.Fatalf("stderr = %q, catalog should have resolved", stderr)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want doctor findings on stdout", stderr)
			}
			if !strings.Contains(stdout, "problem:") {
				t.Fatalf("stdout = %q, want dispatch result", stdout)
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

	t.Run("list combines sorting filtering and limit", func(t *testing.T) {
		status, stdout, stderr := runForTest([]string{"--home", catalogPath, "list", "--tag", "MATH", "--sort", "author", "--limit", "1", "--json"}, nil)
		if status != 0 || stderr != "" {
			t.Fatalf("status = %d, stderr = %q", status, stderr)
		}
		var output []ListOutput
		if err := json.Unmarshal([]byte(stdout), &output); err != nil {
			t.Fatal(err)
		}
		want := []ListOutput{{ID: papers[1].ID, Title: "alpha", Authors: []string{"Alice Smith"}}}
		if !reflect.DeepEqual(output, want) {
			t.Fatalf("list JSON = %#v, want %#v", output, want)
		}
	})

	t.Run("list rejects invalid sort and limit", func(t *testing.T) {
		status, _, stderr := runForTest([]string{"--home", catalogPath, "list", "--sort", "rating"}, nil)
		if status != 1 || !strings.Contains(stderr, "unknown list sort") {
			t.Fatalf("invalid sort: status = %d, stderr = %q", status, stderr)
		}
		status, _, stderr = runForTest([]string{"--home", catalogPath, "list", "--limit", "0"}, nil)
		if status != 2 || !strings.Contains(stderr, "--limit must be a positive integer") {
			t.Fatalf("invalid limit: status = %d, stderr = %q", status, stderr)
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

func TestInfoCommandEmptyAndPopulated(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), "catalog")
	if err := catalog.Initialize(catalogPath, "Mathematics", "Mathematics papers and notes", time.Now()); err != nil {
		t.Fatal(err)
	}

	t.Run("empty through environment", func(t *testing.T) {
		status, stdout, stderr := runForTest([]string{"info"}, map[string]string{"PAPERSPLZ_HOME": catalogPath})
		if status != 0 || stderr != "" {
			t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
		}
		want := "Name: Mathematics\nDescription: Mathematics papers and notes\nPath: " + catalogPath + "\nPapers: 0\nTags: 0\n"
		if stdout != want || strings.Contains(stdout, "Last added:") {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
	})

	firstAt := time.Date(2026, 8, 23, 15, 0, 0, 0, time.UTC)
	lastAt := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	first := cliFixturePaper("aaaa000000000000", "First", nil, []string{"math", "topology"}, firstAt)
	last := cliFixturePaper("bbbb000000000000", "Last", nil, []string{"math", "physics"}, lastAt)
	writeCLIFixturePaper(t, catalogPath, first)
	writeCLIFixturePaper(t, catalogPath, last)

	t.Run("populated JSON and home override", func(t *testing.T) {
		status, stdout, stderr := runForTest(
			[]string{"--home", catalogPath, "info", "--json"},
			map[string]string{"PAPERSPLZ_HOME": filepath.Join(t.TempDir(), "wrong")},
		)
		if status != 0 || stderr != "" {
			t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
		}
		var output InfoOutput
		if err := json.Unmarshal([]byte(stdout), &output); err != nil {
			t.Fatal(err)
		}
		if output.Name != "Mathematics" || output.Description != "Mathematics papers and notes" || output.Path != catalogPath || output.PaperCount != 2 || output.TagCount != 3 || output.LastAdded == nil || !output.LastAdded.Equal(lastAt) {
			t.Fatalf("info JSON = %#v", output)
		}
	})

	t.Run("populated human date", func(t *testing.T) {
		status, stdout, stderr := runForTest([]string{"--home", catalogPath, "info"}, nil)
		if status != 0 || stderr != "" || !strings.Contains(stdout, "Papers: 2\nTags: 3\nLast added: 2026-08-25\n") {
			t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
		}
	})
}

func TestEditCommandUpdatesMetadataWithPrefix(t *testing.T) {
	catalogPath, paper := removeCLIFixture(t)
	paper.Source = "Old"
	paper.SourceURL = "https://old.example/paper"
	recordPath := filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID, store.RecordFilename)
	if err := store.WritePaper(recordPath, paper); err != nil {
		t.Fatal(err)
	}

	status, stdout, stderr := runForTest([]string{
		"--home", catalogPath, "edit", "abcd",
		"--title", "Corrected", "--author", "Bob", "--author", "Carol",
		"--source", "New Journal", "--source-url", "https://new.example/paper",
	}, nil)
	if status != 0 || stderr != "" || !strings.Contains(stdout, paper.ID) {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	stored, err := store.ReadPaper(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Title != "Corrected" || !reflect.DeepEqual(stored.Authors, []string{"Bob", "Carol"}) || stored.Source != "New Journal" || stored.SourceURL != "https://new.example/paper" || stored.ID != paper.ID || !stored.AddedAt.Equal(paper.AddedAt) || !stored.UpdatedAt.After(paper.UpdatedAt) {
		t.Fatalf("stored paper = %#v", stored)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "edit", paper.ID}, nil)
	if status == 0 || stdout != "" || !strings.Contains(stderr, "at least one metadata field") {
		t.Fatalf("empty edit: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestTagsCommandCountsOrderingJSONAndEmptyCatalog(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), "catalog")
	if err := catalog.Initialize(catalogPath, "Tags", "", time.Now()); err != nil {
		t.Fatal(err)
	}

	status, stdout, stderr := runForTest([]string{"--home", catalogPath, "tags"}, nil)
	if status != 0 || stderr != "" || stdout != "No tags.\n" {
		t.Fatalf("empty human: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "tags", "--json"}, nil)
	if status != 0 || stderr != "" || strings.TrimSpace(stdout) != "[]" {
		t.Fatalf("empty JSON: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}

	timestamp := time.Date(2026, 8, 24, 15, 31, 0, 0, time.UTC)
	for _, paper := range []model.Paper{
		cliFixturePaper("aaaa000000000000", "One", nil, []string{"topology", "homotopy", "to-read"}, timestamp),
		cliFixturePaper("bbbb000000000000", "Two", nil, []string{"topology", "algebraic-topology", "to-read"}, timestamp),
		cliFixturePaper("cccc000000000000", "Three", nil, []string{"topology", "algebraic-topology", "homotopy"}, timestamp),
	} {
		writeCLIFixturePaper(t, catalogPath, paper)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "tags"}, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("human: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	wantLines := [][]string{{"topology", "3"}, {"algebraic-topology", "2"}, {"homotopy", "2"}, {"to-read", "2"}}
	if len(lines) != len(wantLines) {
		t.Fatalf("human lines = %q", lines)
	}
	for i := range lines {
		if fields := strings.Fields(lines[i]); !reflect.DeepEqual(fields, wantLines[i]) {
			t.Fatalf("line %d fields = %v, want %v; output = %q", i, fields, wantLines[i], stdout)
		}
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "tags", "--json"}, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("JSON: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	var output []TagUsageOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatal(err)
	}
	want := []TagUsageOutput{{Tag: "topology", Count: 3}, {Tag: "algebraic-topology", Count: 2}, {Tag: "homotopy", Count: 2}, {Tag: "to-read", Count: 2}}
	if !reflect.DeepEqual(output, want) {
		t.Fatalf("JSON = %#v, want %#v", output, want)
	}
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

func TestReviewCommands(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), "catalog")
	if err := catalog.Initialize(catalogPath, "Test", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	paper := cliFixturePaper("abcd000000000000", "Reviewed Paper", nil, nil, time.Now().UTC())
	writeCLIFixturePaper(t, catalogPath, paper)
	recordPath := filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID, store.RecordFilename)

	status, stdout, stderr := runForTest([]string{"--home", catalogPath, "review", "set", "abcd", "Direct review"}, nil)
	if status != 0 || stderr != "" || !strings.Contains(stdout, paper.ID) {
		t.Fatalf("direct set: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "review", "show", "abcd"}, nil)
	if status != 0 || stderr != "" || !strings.Contains(stdout, "Text:\nDirect review\n") || !strings.Contains(stdout, "Created:") {
		t.Fatalf("human show: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "review", "show", "abcd", "--json"}, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("JSON show: status = %d, stderr = %q", status, stderr)
	}
	var reviewOutput ReviewOutput
	if err := json.Unmarshal([]byte(stdout), &reviewOutput); err != nil {
		t.Fatal(err)
	}
	if reviewOutput.Text != "Direct review" || reviewOutput.CreatedAt.IsZero() || reviewOutput.UpdatedAt.IsZero() {
		t.Fatalf("review JSON = %#v", reviewOutput)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &object); err != nil || len(object) != 3 {
		t.Fatalf("review JSON structure = %v, error = %v", object, err)
	}

	reviewFile := filepath.Join(t.TempDir(), "review.txt")
	fileText := "Review from file\nwith another line\n"
	if err := os.WriteFile(reviewFile, []byte(fileText), 0o644); err != nil {
		t.Fatal(err)
	}
	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "review", "set", "abcd", "--file", reviewFile}, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("file set: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	stored, err := store.ReadPaper(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Review == nil || stored.Review.Text != fileText {
		t.Fatalf("file review = %#v", stored.Review)
	}

	stdinText := "Review from stdin\n"
	status, stdout, stderr = runWithInputForTest([]string{"--home", catalogPath, "review", "set", "abcd", "-"}, stdinText, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("stdin set: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	stored, err = store.ReadPaper(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Review == nil || stored.Review.Text != stdinText {
		t.Fatalf("stdin review = %#v", stored.Review)
	}

	before, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "review", "edit", "abcd"}, nil)
	if status == 0 || stdout != "" || !strings.Contains(stderr, "EDITOR is not set") {
		t.Fatalf("unset editor: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	after, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("failed edit changed record.json")
	}

	editorPath := filepath.Join(t.TempDir(), "editor.sh")
	editorScript := "#!/bin/sh\nprintf '%s\\n' 'Edited review' > \"$1\"\n"
	if err := os.WriteFile(editorPath, []byte(editorScript), 0o755); err != nil {
		t.Fatal(err)
	}
	status, stdout, stderr = runWithInputForTest([]string{"--home", catalogPath, "review", "edit", "abcd"}, "", map[string]string{"EDITOR": editorPath})
	if status != 0 || stderr != "" {
		t.Fatalf("edit: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	stored, err = store.ReadPaper(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Review == nil || stored.Review.Text != "Edited review\n" {
		t.Fatalf("edited review = %#v", stored.Review)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "review", "remove", "abcd"}, nil)
	if status != 0 || stderr != "" || !strings.Contains(stdout, paper.ID) {
		t.Fatalf("remove: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	stored, err = store.ReadPaper(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Review != nil {
		t.Fatalf("review after remove = %#v", stored.Review)
	}
}

func TestCommentCommands(t *testing.T) {
	catalogPath, paper := removeCLIFixture(t)

	status, stdout, stderr := runForTest([]string{"--home", catalogPath, "comment", "add", "abcd", "First note"}, nil)
	if status != 0 || stderr != "" || !strings.Contains(stdout, paper.ID) {
		t.Fatalf("add first: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	stored, err := store.ReadPaper(filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID, store.RecordFilename))
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Comments) != 1 || stored.Comments[0].Text != "First note" {
		t.Fatalf("comments after add = %#v", stored.Comments)
	}
	first := stored.Comments[0]

	status, _, stderr = runForTest([]string{"--home", catalogPath, "comment", "add", paper.ID, "Second note"}, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("add second: status = %d, stderr = %q", status, stderr)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "comment", "list", "abcd", "--json"}, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("list JSON: status = %d, stderr = %q", status, stderr)
	}
	var listed []CommentOutput
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatalf("decode list JSON: %v; output = %q", err, stdout)
	}
	if len(listed) != 2 || listed[0].ID != first.ID || listed[0].Text != "First note" {
		t.Fatalf("list JSON = %#v", listed)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "comment", "show", "abcd", first.ID[:8], "--json"}, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("show JSON: status = %d, stderr = %q", status, stderr)
	}
	var shown map[string]any
	if err := json.Unmarshal([]byte(stdout), &shown); err != nil {
		t.Fatal(err)
	}
	if len(shown) != 4 || shown["id"] != first.ID || shown["text"] != "First note" {
		t.Fatalf("show JSON = %#v", shown)
	}

	status, _, stderr = runForTest([]string{"--home", catalogPath, "comment", "edit", "abcd", first.ID[:8]}, nil)
	if status == 0 || !strings.Contains(stderr, "EDITOR is not set") {
		t.Fatalf("edit without editor: status = %d, stderr = %q", status, stderr)
	}
	editorPath := filepath.Join(t.TempDir(), "editor.sh")
	if err := os.WriteFile(editorPath, []byte("#!/bin/sh\nprintf 'Edited note\\n' > \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	status, stdout, stderr = runWithInputForTest([]string{"--home", catalogPath, "comment", "edit", "abcd", first.ID[:8]}, "", map[string]string{"EDITOR": editorPath})
	if status != 0 || stderr != "" || !strings.Contains(stdout, first.ID) {
		t.Fatalf("edit: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	stored, err = store.ReadPaper(filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID, store.RecordFilename))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Comments[0].Text != "Edited note\n" || !stored.Comments[0].CreatedAt.Equal(first.CreatedAt) || !stored.Comments[0].UpdatedAt.After(first.UpdatedAt) {
		t.Fatalf("edited comment = %#v", stored.Comments[0])
	}
	secondID := stored.Comments[1].ID

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "comment", "remove", "abcd", first.ID[:8]}, nil)
	if status != 0 || stderr != "" || !strings.Contains(stdout, first.ID) {
		t.Fatalf("remove: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	stored, err = store.ReadPaper(filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID, store.RecordFilename))
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Comments) != 1 || stored.Comments[0].ID != secondID {
		t.Fatalf("comments after remove = %#v", stored.Comments)
	}
}

func TestSearchCommandHumanJSONAndInterspersedFilters(t *testing.T) {
	catalogPath, paper := removeCLIFixture(t)
	timestamp := paper.UpdatedAt
	paper.Title = "Spectral Methods"
	paper.Source = "Annals"
	paper.Review = &model.Review{Text: "Serre construction", CreatedAt: timestamp, UpdatedAt: timestamp}
	paper.Comments = []model.Comment{{ID: "11110000", Text: "Check convergence", CreatedAt: timestamp, UpdatedAt: timestamp}}
	if err := store.WritePaper(filepath.Join(catalogPath, catalog.PapersDirectory, paper.ID, store.RecordFilename), paper); err != nil {
		t.Fatal(err)
	}
	other := cliFixturePaper("bbbb000000000000", "Algebra Notes", []string{"Bob"}, []string{"algebra"}, timestamp)
	writeCLIFixturePaper(t, catalogPath, other)

	status, stdout, stderr := runForTest([]string{
		"--home", catalogPath, "search", "spectral", "serre", "convergence", "--tag", "TEST", "--author=ALICE",
	}, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("human search: status = %d, stderr = %q", status, stderr)
	}
	if !strings.Contains(stdout, "Spectral Methods") || strings.Contains(stdout, "Algebra Notes") {
		t.Fatalf("human search output = %q", stdout)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "search", "--json", "--tag=test", "annals"}, nil)
	if status != 0 || stderr != "" {
		t.Fatalf("JSON search: status = %d, stderr = %q", status, stderr)
	}
	var output []SearchOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode JSON: %v; output = %q", err, stdout)
	}
	want := []SearchOutput{{ID: paper.ID, Title: "Spectral Methods", Authors: []string{"Alice"}}}
	if !reflect.DeepEqual(output, want) {
		t.Fatalf("JSON output = %#v, want %#v", output, want)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "search", "not-present", "--json"}, nil)
	if status != 0 || stderr != "" || strings.TrimSpace(stdout) != "[]" {
		t.Fatalf("empty JSON search: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestSearchCommandRejectsUnknownOptions(t *testing.T) {
	status, stdout, stderr := runForTest([]string{"--home", t.TempDir(), "search", "term", "--regexp"}, nil)
	if status != 2 || stdout != "" || !strings.Contains(stderr, "unknown option") {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestDoctorCommandCleanAndProblems(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), "catalog")
	if err := catalog.Initialize(catalogPath, "Test", "", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	status, stdout, stderr := runForTest([]string{"--home", catalogPath, "doctor"}, nil)
	if status != 0 || stderr != "" || stdout != "Catalog is healthy.\n" {
		t.Fatalf("clean doctor: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}

	if err := os.WriteFile(filepath.Join(catalogPath, store.CatalogFilename), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "doctor"}, nil)
	if status == 0 || stderr != "" || !strings.Contains(stdout, "problem: catalog.json") || !strings.Contains(stdout, "problem(s) found") {
		t.Fatalf("corrupt doctor: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}

	status, stdout, stderr = runForTest([]string{"--home", catalogPath, "doctor", "extra"}, nil)
	if status != 2 || stdout != "" || !strings.Contains(stderr, "does not accept arguments") {
		t.Fatalf("doctor arguments: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
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

func TestCommandHelpDoesNotRequireCatalog(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{args: []string{"init", "--help"}, want: []string{"Usage: papersplz init", "--description"}},
		{args: []string{"add", "--help"}, want: []string{"Usage: papersplz add", "--title", "--author", "--tag"}},
		{args: []string{"edit", "--help"}, want: []string{"Usage: papersplz edit", "--title", "--author", "--source-url"}},
		{args: []string{"remove", "--help"}, want: []string{"Usage: papersplz remove", "--yes"}},
		{args: []string{"show", "--help"}, want: []string{"Usage: papersplz show", "--json"}},
		{args: []string{"path", "--help"}, want: []string{"Usage: papersplz path"}},
		{args: []string{"list", "--help"}, want: []string{"Usage: papersplz list", "--author", "--sort", "--reverse", "--limit", "--json"}},
		{args: []string{"info", "--help"}, want: []string{"Usage: papersplz info", "--json"}},
		{args: []string{"search", "--help"}, want: []string{"Usage: papersplz search", "--tag", "--json"}},
		{args: []string{"doctor", "--help"}, want: []string{"Usage: papersplz doctor"}},
		{args: []string{"review", "--help"}, want: []string{"Usage: papersplz review", "show PAPER", "set PAPER"}},
		{args: []string{"comment", "-h"}, want: []string{"Usage: papersplz comment", "add PAPER", "edit PAPER"}},
		{args: []string{"tag", "--help"}, want: []string{"Usage: papersplz tag", "add PAPER", "list PAPER"}},
		{args: []string{"tags", "--help"}, want: []string{"Usage: papersplz tags", "count", "--json"}},
		{args: []string{"review", "show", "--help"}, want: []string{"Usage: papersplz review show", "--json"}},
		{args: []string{"comment", "show", "--help"}, want: []string{"Usage: papersplz comment show", "--json"}},
		{args: []string{"tag", "list", "--help"}, want: []string{"Usage: papersplz tag list", "--json"}},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, "_"), func(t *testing.T) {
			status, stdout, stderr := runForTest(tt.args, nil)
			if status != 0 || stderr != "" {
				t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
			}
			for _, wanted := range tt.want {
				if !strings.Contains(stdout, wanted) {
					t.Errorf("stdout = %q, want %q", stdout, wanted)
				}
			}
		})
	}
}
