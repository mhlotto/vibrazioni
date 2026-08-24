package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type opener struct {
	path string
	args []string
}

var openStoredDocument = systemOpenStoredDocument

func systemOpenStoredDocument(documentPath string) error {
	return openDocumentWith(runtime.GOOS, exec.LookPath, func(path string, args ...string) error {
		return exec.Command(path, args...).Run()
	}, documentPath)
}

func openDocumentWith(goos string, lookPath func(string) (string, error), run func(string, ...string) error, documentPath string) error {
	opener, err := selectOpener(goos, lookPath)
	if err != nil {
		return err
	}
	args := append(append([]string{}, opener.args...), documentPath)
	if err := run(opener.path, args...); err != nil {
		return fmt.Errorf("execute %s: %w", opener.path, err)
	}
	return nil
}

func selectOpener(goos string, lookPath func(string) (string, error)) (opener, error) {
	var candidates []opener
	switch goos {
	case "darwin":
		candidates = []opener{{path: "open"}}
	case "linux":
		candidates = []opener{{path: "xdg-open"}}
	case "freebsd":
		candidates = []opener{{path: "xdg-open"}, {path: "gio", args: []string{"open"}}}
	default:
		return opener{}, fmt.Errorf("opening documents is not supported on %s", goos)
	}
	for _, candidate := range candidates {
		path, err := lookPath(candidate.path)
		if err == nil {
			candidate.path = path
			return candidate, nil
		}
	}
	names := make([]string, len(candidates))
	for i := range candidates {
		names[i] = candidates[i].path
	}
	return opener{}, fmt.Errorf("no document opener found for %s (tried %s)", goos, strings.Join(names, ", "))
}
