package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func TestAddLocalCopiesAndRecordsDocument(t *testing.T) {
	catalogPath := newTestCatalog(t)
	sourcePath := filepath.Join(t.TempDir(), "Interesting.PDF")
	contents := []byte("independent paper contents\n")
	if err := os.WriteFile(sourcePath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	addedAt := time.Date(2026, 8, 24, 15, 31, 0, 0, time.FixedZone("EDT", -4*60*60))

	paper, err := AddLocal(catalogPath, sourcePath, AddOptions{
		Title:   "  Some Interesting Paper  ",
		Authors: []string{" Alice Smith ", "Bob Jones"},
		Source:  " Journal of Interesting Things ",
		Tags:    []string{" Topology ", "homotopy", "topology"},
	}, addedAt)
	if err != nil {
		t.Fatalf("AddLocal() error = %v", err)
	}
	if len(paper.ID) != 32 {
		t.Fatalf("paper id = %q, want 32 characters", paper.ID)
	}
	if paper.Title != "Some Interesting Paper" {
		t.Fatalf("title = %q", paper.Title)
	}
	if !reflect.DeepEqual(paper.Authors, []string{"Alice Smith", "Bob Jones"}) {
		t.Fatalf("authors = %#v", paper.Authors)
	}
	if !reflect.DeepEqual(paper.Tags, []string{"topology", "homotopy"}) {
		t.Fatalf("tags = %#v", paper.Tags)
	}
	if paper.Source != "Journal of Interesting Things" {
		t.Fatalf("source = %q", paper.Source)
	}
	if paper.AddedAt != addedAt.UTC() || paper.UpdatedAt != addedAt.UTC() {
		t.Fatalf("timestamps = %v, %v", paper.AddedAt, paper.UpdatedAt)
	}
	if paper.File.Name != "paper.pdf" || paper.File.OriginalName != "Interesting.PDF" {
		t.Fatalf("file metadata = %#v", paper.File)
	}
	if paper.File.Size != int64(len(contents)) {
		t.Fatalf("file size = %d, want %d", paper.File.Size, len(contents))
	}
	wantHash := sha256.Sum256(contents)
	if paper.File.SHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("sha256 = %q", paper.File.SHA256)
	}

	recordPath := filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename)
	storedRecord, err := store.ReadPaper(recordPath)
	if err != nil {
		t.Fatalf("ReadPaper() error = %v", err)
	}
	if !reflect.DeepEqual(storedRecord, paper) {
		t.Fatalf("stored record = %#v, want %#v", storedRecord, paper)
	}
	storedPath := filepath.Join(catalogPath, PapersDirectory, paper.ID, paper.File.Name)
	storedContents, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(storedContents) != string(contents) {
		t.Fatalf("stored contents = %q, want %q", storedContents, contents)
	}

	if err := os.WriteFile(sourcePath, []byte("source changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	storedContents, err = os.ReadFile(storedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(storedContents) != string(contents) {
		t.Fatal("stored copy changed when source changed")
	}
}

func TestAddLocalFallsBackToBinExtension(t *testing.T) {
	catalogPath := newTestCatalog(t)
	sourcePath := filepath.Join(t.TempDir(), "paper.strange-extension-that-is-too-long")
	if err := os.WriteFile(sourcePath, []byte("paper"), 0o644); err != nil {
		t.Fatal(err)
	}
	paper, err := AddLocal(catalogPath, sourcePath, AddOptions{Title: "Fallback"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if paper.File.Name != "paper.bin" {
		t.Fatalf("stored filename = %q, want paper.bin", paper.File.Name)
	}
}

func TestAddLocalRejectsDuplicateContent(t *testing.T) {
	catalogPath := newTestCatalog(t)
	sourceDirectory := t.TempDir()
	firstPath := filepath.Join(sourceDirectory, "first.pdf")
	secondPath := filepath.Join(sourceDirectory, "second.ps")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte("same content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	first, err := AddLocal(catalogPath, firstPath, AddOptions{Title: "First"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = AddLocal(catalogPath, secondPath, AddOptions{Title: "Second"}, time.Now())
	if !errors.Is(err, ErrDuplicateContent) {
		t.Fatalf("AddLocal() error = %v, want ErrDuplicateContent", err)
	}
	var duplicate *DuplicateContentError
	if !errors.As(err, &duplicate) || duplicate.PaperID != first.ID {
		t.Fatalf("duplicate error = %#v, want existing id %s", duplicate, first.ID)
	}
	entries, err := os.ReadDir(filepath.Join(catalogPath, PapersDirectory))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != first.ID {
		t.Fatalf("paper directories = %v, want only %s", entryNames(entries), first.ID)
	}
	assertNoImportStages(t, catalogPath)
}

func TestAddLocalFailureLeavesNoPartialPaper(t *testing.T) {
	catalogPath := newTestCatalog(t)
	sourcePath := filepath.Join(t.TempDir(), "paper.pdf")
	if err := os.WriteFile(sourcePath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := AddLocal(catalogPath, sourcePath, AddOptions{
		Title: "Bad Tag",
		Tags:  []string{"not valid"},
	}, time.Now())
	if err == nil || !strings.Contains(err.Error(), "invalid tag") {
		t.Fatalf("AddLocal() error = %v, want invalid tag", err)
	}
	entries, err := os.ReadDir(filepath.Join(catalogPath, PapersDirectory))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed import left paper directories: %v", entryNames(entries))
	}
	assertNoImportStages(t, catalogPath)
}

func newTestCatalog(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "catalog")
	if err := Initialize(path, "Test Catalog", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertNoImportStages(t *testing.T, catalogPath string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(catalogPath, ".papersplz-import-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("import staging directories left behind: %v", matches)
	}
}
