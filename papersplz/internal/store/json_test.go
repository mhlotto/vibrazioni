package store

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

func testCatalog() model.Catalog {
	return model.Catalog{
		SchemaVersion: model.SchemaVersion,
		Name:          "Mathematics",
		Description:   "Papers and notes",
		CreatedAt:     time.Date(2026, 8, 24, 15, 30, 0, 0, time.UTC),
	}
}

func testPaper() model.Paper {
	timestamp := time.Date(2026, 8, 24, 15, 31, 0, 0, time.UTC)
	return model.Paper{
		SchemaVersion: model.SchemaVersion,
		ID:            "a81f32c991b7",
		Title:         "Some Interesting Paper",
		Authors:       []string{"Alice Smith", "Bob Jones"},
		Source:        "Journal of Interesting Things",
		SourceURL:     "https://example.org/paper.pdf",
		AddedAt:       timestamp,
		UpdatedAt:     timestamp,
		File: model.File{
			Name:         "paper.pdf",
			OriginalName: "smith-interesting-paper.pdf",
			Size:         481231,
			SHA256:       strings.Repeat("a", 64),
		},
		Tags:     []string{"topology", "homotopy"},
		Review:   nil,
		Comments: []model.Comment{},
	}
}

func TestCatalogRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), CatalogFilename)
	want := testCatalog()
	if err := WriteCatalog(path, want); err != nil {
		t.Fatalf("WriteCatalog() error = %v", err)
	}
	got, err := ReadCatalog(path)
	if err != nil {
		t.Fatalf("ReadCatalog() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadCatalog() = %#v, want %#v", got, want)
	}
	assertReadableJSON(t, path)
}

func TestPaperRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), RecordFilename)
	want := testPaper()
	if err := WritePaper(path, want); err != nil {
		t.Fatalf("WritePaper() error = %v", err)
	}
	got, err := ReadPaper(path)
	if err != nil {
		t.Fatalf("ReadPaper() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadPaper() = %#v, want %#v", got, want)
	}
	assertReadableJSON(t, path)
}

func TestReadRejectsMalformedAndStructurallyInvalidJSON(t *testing.T) {
	directory := t.TempDir()
	tests := []struct {
		name string
		data string
	}{
		{name: "malformed", data: `{"schema_version": 1`},
		{name: "multiple values", data: `{}` + "\n" + `{}`},
		{name: "missing required fields", data: `{"schema_version":1}`},
		{name: "bad timestamp", data: `{"schema_version":1,"name":"Catalog","created_at":"yesterday"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(directory, strings.ReplaceAll(tt.name, " ", "-")+".json")
			if err := os.WriteFile(path, []byte(tt.data), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadCatalog(path); err == nil {
				t.Fatal("ReadCatalog() accepted invalid JSON")
			}
		})
	}
}

func TestUnsupportedSchemaDoesNotReplaceExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), CatalogFilename)
	original := []byte("original metadata\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	catalog := testCatalog()
	catalog.SchemaVersion = model.SchemaVersion + 1
	err := WriteCatalog(path, catalog)
	if !errors.Is(err, model.ErrUnsupportedSchema) {
		t.Fatalf("WriteCatalog() error = %v, want ErrUnsupportedSchema", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("file after failed write = %q, want %q", got, original)
	}
}

func TestReadRejectsUnsupportedSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), CatalogFilename)
	data := `{"schema_version":2,"name":"Future","created_at":"2026-08-24T15:30:00Z"}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadCatalog(path)
	if !errors.Is(err, model.ErrUnsupportedSchema) {
		t.Fatalf("ReadCatalog() error = %v, want ErrUnsupportedSchema", err)
	}
}

func TestAtomicReplaceFailureLeavesExistingFileIntact(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, CatalogFilename)
	original := []byte("complete original metadata\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	wantErr := errors.New("simulated write failure")
	err := atomicReplace(path, func(writer io.Writer) error {
		if _, err := io.WriteString(writer, "partial replacement"); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("atomicReplace() error = %v, want simulated failure", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("file after failed replacement = %q, want %q", got, original)
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".papersplz-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func assertReadableJSON(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "\n") || !strings.Contains(string(data), "\n  \"") {
		t.Fatalf("metadata is not indented, newline-terminated JSON:\n%s", data)
	}
}
