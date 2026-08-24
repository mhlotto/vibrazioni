package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV1EndToEndPortableCatalogLifecycle(t *testing.T) {
	workspace := t.TempDir()
	originalCatalog := filepath.Join(workspace, "original-catalog")
	localSource := filepath.Join(workspace, "outside-source.pdf")
	localContents := []byte("local paper contents")
	if err := os.WriteFile(localSource, localContents, 0o644); err != nil {
		t.Fatal(err)
	}

	runE2E(t, []string{"init", originalCatalog, "--name", "Acceptance", "--description", "Portable v1 catalog"}, nil)
	localAdd := runE2E(t, []string{
		"--home", originalCatalog, "add", localSource, "--title", "Local Topology",
		"--author", "Alice Example", "--source", "Proceedings", "--tag", "Topology",
	}, nil)
	localID := e2eField(t, localAdd, 1)

	remoteContents := []byte("remote paper contents")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/remote.ps" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/postscript")
		_, _ = writer.Write(remoteContents)
	}))
	remoteAdd := runE2E(t, []string{
		"--home", originalCatalog, "add", server.URL + "/remote.ps", "--title", "Remote Algebra", "--author", "Bob Example",
	}, nil)
	remoteID := e2eField(t, remoteAdd, 1)

	listHuman := runE2E(t, []string{"--home", originalCatalog, "list"}, nil)
	if !strings.Contains(listHuman, "Local Topology") || !strings.Contains(listHuman, "Remote Algebra") {
		t.Fatalf("list output = %q", listHuman)
	}
	showHuman := runE2E(t, []string{"--home", originalCatalog, "show", localID[:8]}, nil)
	if !strings.Contains(showHuman, "Title: Local Topology") || !strings.Contains(showHuman, "Tags: topology") {
		t.Fatalf("show output = %q", showHuman)
	}
	assertE2EJSON(t, runE2E(t, []string{"--home", originalCatalog, "list", "--json"}, nil), &[]ListOutput{})
	assertE2EJSON(t, runE2E(t, []string{"--home", originalCatalog, "show", localID, "--json"}, nil), &ShowOutput{})

	localPath := strings.TrimSpace(runE2E(t, []string{"--home", originalCatalog, "path", localID}, nil))
	if !strings.HasPrefix(localPath, originalCatalog+string(os.PathSeparator)) {
		t.Fatalf("path output %q is outside original catalog", localPath)
	}
	if contents, err := os.ReadFile(localPath); err != nil || string(contents) != string(localContents) {
		t.Fatalf("stored local document = %q, %v", contents, err)
	}

	runE2E(t, []string{"--home", originalCatalog, "tag", "add", localID[:8], "To-Read", "geometry"}, nil)
	runE2E(t, []string{"--home", originalCatalog, "tag", "remove", localID, "geometry"}, nil)
	var tags TagListOutput
	assertE2EJSON(t, runE2E(t, []string{"--home", originalCatalog, "tag", "list", localID, "--json"}, nil), &tags)
	if strings.Join(tags.Tags, ",") != "topology,to-read" {
		t.Fatalf("tags = %v", tags.Tags)
	}

	runE2E(t, []string{"--home", originalCatalog, "review", "set", localID, "Initial spectral review"}, nil)
	editor := filepath.Join(workspace, "editor.sh")
	if err := os.WriteFile(editor, []byte("#!/bin/sh\nprintf 'Edited spectral review\\n' > \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runE2EWithInput(t, []string{"--home", originalCatalog, "review", "edit", localID}, "", map[string]string{"EDITOR": editor})
	var review ReviewOutput
	assertE2EJSON(t, runE2E(t, []string{"--home", originalCatalog, "review", "show", localID, "--json"}, nil), &review)
	if review.Text != "Edited spectral review\n" {
		t.Fatalf("review text = %q", review.Text)
	}

	commentAdd := runE2E(t, []string{"--home", originalCatalog, "comment", "add", localID, "Check Lemma 4"}, nil)
	commentID := e2eField(t, commentAdd, 2)
	if err := os.WriteFile(editor, []byte("#!/bin/sh\nprintf 'Edited convergence note\\n' > \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runE2EWithInput(t, []string{"--home", originalCatalog, "comment", "edit", localID, commentID[:8]}, "", map[string]string{"EDITOR": editor})
	var comments []CommentOutput
	assertE2EJSON(t, runE2E(t, []string{"--home", originalCatalog, "comment", "list", localID, "--json"}, nil), &comments)
	if len(comments) != 1 || comments[0].Text != "Edited convergence note\n" {
		t.Fatalf("comments = %#v", comments)
	}
	var comment CommentOutput
	assertE2EJSON(t, runE2E(t, []string{"--home", originalCatalog, "comment", "show", localID, commentID[:8], "--json"}, nil), &comment)
	if comment.ID != commentID {
		t.Fatalf("shown comment ID = %q, want %q", comment.ID, commentID)
	}

	var search []SearchOutput
	assertE2EJSON(t, runE2E(t, []string{
		"--home", originalCatalog, "search", "spectral", "convergence", "--tag", "topology", "--author", "alice", "--json",
	}, nil), &search)
	if len(search) != 1 || search[0].ID != localID {
		t.Fatalf("search results = %#v", search)
	}

	runE2E(t, []string{"--home", originalCatalog, "comment", "remove", localID, commentID[:8]}, nil)
	server.Close()
	if err := os.Remove(localSource); err != nil {
		t.Fatal(err)
	}
	movedCatalog := filepath.Join(workspace, "moved", "catalog")
	copyE2ETree(t, originalCatalog, movedCatalog)

	listFromEnvironment := runE2E(t, []string{"list"}, map[string]string{"PAPERSPLZ_HOME": movedCatalog})
	if !strings.Contains(listFromEnvironment, "Local Topology") || !strings.Contains(listFromEnvironment, "Remote Algebra") {
		t.Fatalf("moved environment list = %q", listFromEnvironment)
	}
	listFromFlag := runE2E(t, []string{"--home", movedCatalog, "list"}, map[string]string{"PAPERSPLZ_HOME": filepath.Join(workspace, "wrong")})
	if listFromFlag != listFromEnvironment {
		t.Fatalf("--home did not override environment:\nflag: %q\nenv:  %q", listFromFlag, listFromEnvironment)
	}
	movedPath := strings.TrimSpace(runE2E(t, []string{"--home", movedCatalog, "path", localID}, nil))
	if !strings.HasPrefix(movedPath, movedCatalog+string(os.PathSeparator)) || strings.Contains(movedPath, originalCatalog) {
		t.Fatalf("moved path = %q", movedPath)
	}
	if contents, err := os.ReadFile(movedPath); err != nil || string(contents) != string(localContents) {
		t.Fatalf("moved local document = %q, %v", contents, err)
	}
	remotePath := strings.TrimSpace(runE2E(t, []string{"--home", movedCatalog, "path", remoteID}, nil))
	if contents, err := os.ReadFile(remotePath); err != nil || string(contents) != string(remoteContents) {
		t.Fatalf("moved remote document = %q, %v", contents, err)
	}
	if doctor := runE2E(t, []string{"--home", movedCatalog, "doctor"}, nil); doctor != "Catalog is healthy.\n" {
		t.Fatalf("doctor output = %q", doctor)
	}
	runE2E(t, []string{"--home", movedCatalog, "remove", remoteID[:8], "--yes"}, nil)
	remaining := runE2E(t, []string{"--home", movedCatalog, "list"}, nil)
	if !strings.Contains(remaining, "Local Topology") || strings.Contains(remaining, "Remote Algebra") {
		t.Fatalf("list after removal = %q", remaining)
	}
	if doctor := runE2E(t, []string{"--home", movedCatalog, "doctor"}, nil); doctor != "Catalog is healthy.\n" {
		t.Fatalf("doctor after removal = %q", doctor)
	}
}

func runE2E(t *testing.T, args []string, environment map[string]string) string {
	t.Helper()
	status, stdout, stderr := runForTest(args, environment)
	if status != 0 {
		t.Fatalf("papersplz %s: status = %d, stdout = %q, stderr = %q", strings.Join(args, " "), status, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("papersplz %s: unexpected stderr = %q", strings.Join(args, " "), stderr)
	}
	return stdout
}

func runE2EWithInput(t *testing.T, args []string, input string, environment map[string]string) string {
	t.Helper()
	status, stdout, stderr := runWithInputForTest(args, input, environment)
	if status != 0 || stderr != "" {
		t.Fatalf("papersplz %s: status = %d, stdout = %q, stderr = %q", strings.Join(args, " "), status, stdout, stderr)
	}
	return stdout
}

func e2eField(t *testing.T, output string, index int) string {
	t.Helper()
	fields := strings.Fields(output)
	if index >= len(fields) {
		t.Fatalf("output %q has no field %d", output, index)
	}
	return strings.TrimSuffix(fields[index], ".")
}

func assertE2EJSON(t *testing.T, output string, destination any) {
	t.Helper()
	if err := json.Unmarshal([]byte(output), destination); err != nil {
		t.Fatalf("invalid JSON output %q: %v", output, err)
	}
}

func copyE2ETree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, contents, info.Mode().Perm()); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(fmt.Errorf("copy catalog: %w", err))
	}
}
