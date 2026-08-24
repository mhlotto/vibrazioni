package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func TestDoctorCleanCatalogIsReadOnly(t *testing.T) {
	catalogPath := newTestCatalog(t)
	writeDoctorPaper(t, catalogPath, doctorPaper("aaaa00000000", "Clean", []byte("clean document")))
	before := snapshotDoctorTree(t, catalogPath)

	problems := Doctor(catalogPath)
	if len(problems) != 0 {
		t.Fatalf("Doctor() problems = %v", problems)
	}
	after := snapshotDoctorTree(t, catalogPath)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("doctor mutated catalog\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func TestDoctorReportsMultipleIndependentProblems(t *testing.T) {
	catalogPath := newTestCatalog(t)
	if err := os.WriteFile(filepath.Join(catalogPath, store.CatalogFilename), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}

	first := doctorPaper("abcd0000", "First", []byte("duplicate bytes"))
	first.Tags = []string{"Bad Tag"}
	writeDoctorPaperRaw(t, catalogPath, "aaaa0000", first, []byte("wrong"))
	if err := os.WriteFile(filepath.Join(catalogPath, PapersDirectory, "aaaa0000", "extra.tmp"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	second := doctorPaper("abcd0000", "Second", []byte("duplicate bytes"))
	second.SchemaVersion = 2
	writeDoctorPaperRaw(t, catalogPath, "bbbb0000", second, []byte("duplicate bytes"))

	badReview := doctorPaper("cccc0000", "Bad Review", []byte("review"))
	badReview.Review = &model.Review{}
	writeDoctorPaperRaw(t, catalogPath, badReview.ID, badReview, []byte("review"))

	badComment := doctorPaper("dddd0000", "Bad Comment", []byte("comment"))
	badComment.Comments = []model.Comment{{ID: "not-hex", Text: "", CreatedAt: time.Time{}, UpdatedAt: time.Time{}}}
	writeDoctorPaperRaw(t, catalogPath, badComment.ID, badComment, []byte("comment"))

	missing := doctorPaper("eeee0000", "Missing", []byte("missing"))
	writeDoctorPaperRaw(t, catalogPath, missing.ID, missing, nil)
	if err := os.Mkdir(filepath.Join(catalogPath, PapersDirectory, "ffff0000"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(catalogPath, PapersDirectory, "loose-file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	problems := Doctor(catalogPath)
	joined := doctorProblemText(problems)
	for _, wanted := range []string{
		"catalog.json: malformed JSON",
		"invalid tag",
		"size is",
		"SHA-256 is",
		"unexpected entry in paper directory",
		"unsupported schema version",
		"paper ID \"abcd0000\" does not match directory name",
		"review text is required",
		"comment 0",
		"referenced document is unavailable",
		"record.json: open",
		"unexpected entry; expected a paper directory",
		"duplicate paper ID abcd0000",
		"duplicate content SHA-256",
	} {
		if !strings.Contains(joined, wanted) {
			t.Errorf("Doctor() output missing %q:\n%s", wanted, joined)
		}
	}
	if len(problems) < 14 {
		t.Fatalf("Doctor() returned only %d problems:\n%s", len(problems), joined)
	}
}

func TestDoctorReportsMissingPapersDirectoryAndInvalidCatalog(t *testing.T) {
	path := t.TempDir()
	problems := Doctor(path)
	joined := doctorProblemText(problems)
	if !strings.Contains(joined, "catalog.json") || !strings.Contains(joined, "papers") || len(problems) != 2 {
		t.Fatalf("Doctor() problems = %v", problems)
	}
}

func TestDoctorReportsUnsupportedCatalogSchema(t *testing.T) {
	path := t.TempDir()
	metadata := model.Catalog{
		SchemaVersion: 2,
		Name:          "Future",
		CreatedAt:     time.Date(2026, 8, 24, 15, 31, 0, 0, time.UTC),
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, store.CatalogFilename), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(path, PapersDirectory), 0o755); err != nil {
		t.Fatal(err)
	}
	problems := Doctor(path)
	if len(problems) != 1 || !strings.Contains(problems[0].String(), "unsupported schema version: 2") {
		t.Fatalf("Doctor() problems = %v", problems)
	}
}

func TestDoctorReportsKnownTemporaryArtifactsWithoutRemovingThem(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := doctorPaper("aaaa00000000", "Clean", []byte("clean document"))
	writeDoctorPaper(t, catalogPath, paper)
	artifacts := []string{
		filepath.Join(catalogPath, ".papersplz-import-stale"),
		filepath.Join(catalogPath, ".papersplz-123.tmp"),
		filepath.Join(catalogPath, PapersDirectory, paper.ID, ".papersplz-456.tmp"),
	}
	if err := os.Mkdir(artifacts[0], 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range artifacts[1:] {
		if err := os.WriteFile(path, []byte("partial"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	unknownHidden := filepath.Join(catalogPath, ".user-note")
	if err := os.WriteFile(unknownHidden, []byte("leave me alone"), 0o600); err != nil {
		t.Fatal(err)
	}

	before := snapshotDoctorTree(t, catalogPath)
	problems := Doctor(catalogPath)
	joined := doctorProblemText(problems)
	for _, wanted := range []string{
		".papersplz-import-stale: abandoned papersplz import staging artifact",
		".papersplz-123.tmp: abandoned papersplz temporary metadata artifact",
		filepath.Join(PapersDirectory, paper.ID, ".papersplz-456.tmp") + ": abandoned papersplz temporary metadata artifact",
	} {
		if !strings.Contains(joined, wanted) {
			t.Errorf("Doctor() output missing %q:\n%s", wanted, joined)
		}
	}
	if strings.Contains(joined, ".user-note") {
		t.Fatalf("Doctor() classified unrelated hidden file:\n%s", joined)
	}
	after := snapshotDoctorTree(t, catalogPath)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("doctor mutated temporary state\nbefore: %#v\nafter: %#v", before, after)
	}
}

func doctorPaper(id, title string, document []byte) model.Paper {
	timestamp := time.Date(2026, 8, 24, 15, 31, 0, 0, time.UTC)
	digest := sha256.Sum256(document)
	return model.Paper{
		SchemaVersion: model.SchemaVersion,
		ID:            id,
		Title:         title,
		Authors:       []string{},
		AddedAt:       timestamp,
		UpdatedAt:     timestamp,
		File: model.File{
			Name:         "paper.pdf",
			OriginalName: "original.pdf",
			Size:         int64(len(document)),
			SHA256:       hex.EncodeToString(digest[:]),
		},
		Tags:     []string{},
		Comments: []model.Comment{},
	}
}

func writeDoctorPaper(t *testing.T, catalogPath string, paper model.Paper) {
	t.Helper()
	document := []byte("clean document")
	writeDoctorPaperRaw(t, catalogPath, paper.ID, paper, document)
}

func writeDoctorPaperRaw(t *testing.T, catalogPath, directory string, paper model.Paper, document []byte) {
	t.Helper()
	directoryPath := filepath.Join(catalogPath, PapersDirectory, directory)
	if err := os.Mkdir(directoryPath, 0o755); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(paper, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(directoryPath, store.RecordFilename), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if document != nil {
		if err := os.WriteFile(filepath.Join(directoryPath, paper.File.Name), document, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func doctorProblemText(problems []DoctorProblem) string {
	lines := make([]string, len(problems))
	for i, problem := range problems {
		lines[i] = problem.String()
	}
	return strings.Join(lines, "\n")
}

func snapshotDoctorTree(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := make(map[string]string)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			snapshot[relative] = "directory"
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[relative] = string(contents)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
