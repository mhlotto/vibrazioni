package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		{name: "home flag", args: []string{"--home", "/flag/catalog", "list"}},
		{name: "environment", args: []string{"list"}, env: map[string]string{"PAPERSPLZ_HOME": "/env/catalog"}},
		{name: "flag overrides environment", args: []string{"--home", "/flag/catalog", "list"}, env: map[string]string{"PAPERSPLZ_HOME": "/env/catalog"}},
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
