package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
)

func TestResolvePaperID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog")
	if err := Initialize(path, "Test", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a81f32c991b7", "a81f99bb0011", "410cbe130000"} {
		if err := os.Mkdir(filepath.Join(path, PapersDirectory, id), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name     string
		selector string
		want     string
		wantErr  error
	}{
		{name: "exact", selector: "a81f32c991b7", want: "a81f32c991b7"},
		{name: "unique prefix", selector: "410c", want: "410cbe130000"},
		{name: "ambiguous", selector: "a81f", wantErr: identity.ErrAmbiguous},
		{name: "missing", selector: "ffff", wantErr: identity.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolvePaperID(path, tt.selector)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolvePaperID() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ResolvePaperID() = %q, want %q", got, tt.want)
			}
		})
	}
}
