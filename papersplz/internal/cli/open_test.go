package cli

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSelectOpenerByPlatform(t *testing.T) {
	tests := []struct {
		name      string
		goos      string
		available map[string]string
		want      opener
	}{
		{name: "macOS", goos: "darwin", available: map[string]string{"open": "/usr/bin/open"}, want: opener{path: "/usr/bin/open"}},
		{name: "Linux", goos: "linux", available: map[string]string{"xdg-open": "/usr/bin/xdg-open"}, want: opener{path: "/usr/bin/xdg-open"}},
		{name: "FreeBSD prefers xdg-open", goos: "freebsd", available: map[string]string{"xdg-open": "/usr/local/bin/xdg-open", "gio": "/usr/local/bin/gio"}, want: opener{path: "/usr/local/bin/xdg-open"}},
		{name: "FreeBSD falls back to gio", goos: "freebsd", available: map[string]string{"gio": "/usr/local/bin/gio"}, want: opener{path: "/usr/local/bin/gio", args: []string{"open"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectOpener(tt.goos, func(name string) (string, error) {
				if path, ok := tt.available[name]; ok {
					return path, nil
				}
				return "", errors.New("not found")
			})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("selectOpener() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSelectOpenerFailures(t *testing.T) {
	missing := func(string) (string, error) { return "", errors.New("not found") }
	if _, err := selectOpener("linux", missing); err == nil || !strings.Contains(err.Error(), "no document opener found") {
		t.Fatalf("missing opener error = %v", err)
	}
	if _, err := selectOpener("plan9", missing); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unsupported platform error = %v", err)
	}
}

func TestOpenDocumentWithExecutesSelectedOpener(t *testing.T) {
	var gotPath string
	var gotArgs []string
	err := openDocumentWith(
		"freebsd",
		func(name string) (string, error) {
			if name == "gio" {
				return "/test/gio", nil
			}
			return "", errors.New("not found")
		},
		func(path string, args ...string) error {
			gotPath = path
			gotArgs = append([]string{}, args...)
			return nil
		},
		"/catalog/papers/abcd/paper.pdf",
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/test/gio" || !reflect.DeepEqual(gotArgs, []string{"open", "/catalog/papers/abcd/paper.pdf"}) {
		t.Fatalf("runner called with path %q, args %q", gotPath, gotArgs)
	}

	wantErr := errors.New("viewer failed")
	err = openDocumentWith("darwin", func(string) (string, error) { return "/usr/bin/open", nil }, func(string, ...string) error {
		return wantErr
	}, "/paper.pdf")
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "execute /usr/bin/open") {
		t.Fatalf("execution error = %v", err)
	}
}
