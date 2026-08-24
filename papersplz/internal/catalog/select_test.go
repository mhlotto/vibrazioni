package catalog

import (
	"errors"
	"testing"
)

func TestResolveHome(t *testing.T) {
	tests := []struct {
		name     string
		explicit string
		env      string
		want     string
		wantErr  error
	}{
		{name: "explicit overrides environment", explicit: "/flags/catalog", env: "/env/catalog", want: "/flags/catalog"},
		{name: "environment fallback", env: "/env/catalog", want: "/env/catalog"},
		{name: "missing", wantErr: ErrHomeRequired},
		{name: "empty environment is missing", env: "", wantErr: ErrHomeRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := func(key string) (string, bool) {
				if key != HomeEnvironment || tt.env == "" {
					return "", false
				}
				return tt.env, true
			}
			got, err := ResolveHome(tt.explicit, lookup)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolveHome() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ResolveHome() = %q, want %q", got, tt.want)
			}
		})
	}
}
