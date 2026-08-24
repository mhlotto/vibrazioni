package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestAddDirectURL(t *testing.T) {
	contents := []byte("downloaded paper contents\n")
	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/pdf")
		response.Write(contents)
	})
	tests := []struct {
		name      string
		newServer func(http.Handler) *httptest.Server
	}{
		{name: "http", newServer: httptest.NewServer},
		{name: "https", newServer: httptest.NewTLSServer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalogPath := newTestCatalog(t)
			server := tt.newServer(handler)
			defer server.Close()
			suppliedURL := server.URL + "/papers/Interesting.PDF?download=1"
			paper, err := AddWithHTTPClient(catalogPath, suppliedURL, AddOptions{
				Title: "Remote Paper",
			}, time.Now(), server.Client())
			if err != nil {
				t.Fatalf("AddWithHTTPClient() error = %v", err)
			}
			if paper.SourceURL != suppliedURL {
				t.Fatalf("source_url = %q, want %q", paper.SourceURL, suppliedURL)
			}
			if paper.File.Name != "paper.pdf" || paper.File.OriginalName != "Interesting.PDF" {
				t.Fatalf("file metadata = %#v", paper.File)
			}
			stored, err := os.ReadFile(filepath.Join(catalogPath, PapersDirectory, paper.ID, paper.File.Name))
			if err != nil {
				t.Fatal(err)
			}
			if string(stored) != string(contents) {
				t.Fatalf("stored contents = %q, want %q", stored, contents)
			}
		})
	}
}

func TestAddURLWithoutFilenameUsesBin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Write([]byte("document"))
	}))
	defer server.Close()
	paper, err := Add(newTestCatalog(t), server.URL+"/", AddOptions{Title: "No Filename"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if paper.File.Name != "paper.bin" || paper.File.OriginalName != "download" {
		t.Fatalf("file metadata = %#v", paper.File)
	}
}

func TestURLImportParticipatesInDuplicateDetection(t *testing.T) {
	catalogPath := newTestCatalog(t)
	contents := []byte("duplicate across source types")
	localPath := filepath.Join(t.TempDir(), "local.pdf")
	if err := os.WriteFile(localPath, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	local, err := AddLocal(catalogPath, localPath, AddOptions{Title: "Local"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Write(contents)
	}))
	defer server.Close()
	_, err = Add(catalogPath, server.URL+"/remote.ps", AddOptions{Title: "Remote"}, time.Now())
	var duplicate *DuplicateContentError
	if !errors.As(err, &duplicate) || duplicate.PaperID != local.ID {
		t.Fatalf("Add() error = %v, want duplicate of %s", err, local.ID)
	}
	assertOnlyPaper(t, catalogPath, local.ID)
	assertNoImportStages(t, catalogPath)
}

func TestHTTPFailuresLeaveNoCompletePaper(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "non-success status",
			handler: func(response http.ResponseWriter, request *http.Request) {
				http.Error(response, "not found", http.StatusNotFound)
			},
		},
		{
			name: "interrupted body",
			handler: func(response http.ResponseWriter, request *http.Request) {
				response.Header().Set("Content-Length", "100")
				response.WriteHeader(http.StatusOK)
				response.Write([]byte("short"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalogPath := newTestCatalog(t)
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			_, err := Add(catalogPath, server.URL+"/paper.pdf", AddOptions{Title: "Failure"}, time.Now())
			if err == nil {
				t.Fatal("Add() succeeded, want download failure")
			}
			entries, readErr := os.ReadDir(filepath.Join(catalogPath, PapersDirectory))
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("failed download left paper directories: %v", entryNames(entries))
			}
			assertNoImportStages(t, catalogPath)
		})
	}
}

func TestHTTPDownloadTimeoutCleansStagingData(t *testing.T) {
	catalogPath := newTestCatalog(t)
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
		if flusher, ok := response.(http.Flusher); ok {
			flusher.Flush()
		}
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()
	client := server.Client()
	client.Timeout = 50 * time.Millisecond

	_, err := AddWithHTTPClient(catalogPath, server.URL+"/stalled.pdf", AddOptions{Title: "Stalled"}, time.Now(), client)
	if err == nil {
		t.Fatal("AddWithHTTPClient() succeeded for stalled response")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(strings.ToLower(err.Error()), "timeout") {
		t.Fatalf("timeout error = %v", err)
	}
	select {
	case <-started:
	default:
		t.Fatal("server response did not start")
	}
	entries, readErr := os.ReadDir(filepath.Join(catalogPath, PapersDirectory))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("timed out download left paper directories: %v", entryNames(entries))
	}
	assertNoImportStages(t, catalogPath)
}

func TestDownloadHTTPClientAppliesDefaultWithoutMutatingCaller(t *testing.T) {
	caller := &http.Client{}
	got := downloadHTTPClient(caller)
	if got == caller || got.Timeout != RemoteDownloadTimeout {
		t.Fatalf("downloadHTTPClient() = %#v", got)
	}
	if caller.Timeout != 0 {
		t.Fatalf("caller timeout changed to %v", caller.Timeout)
	}
	short := &http.Client{Timeout: time.Second}
	if got := downloadHTTPClient(short); got != short {
		t.Fatal("client with finite timeout was replaced")
	}
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

func assertOnlyPaper(t *testing.T, catalogPath, id string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(catalogPath, PapersDirectory))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != id {
		t.Fatalf("paper directories = %v, want only %s", entryNames(entries), id)
	}
}
