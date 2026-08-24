package identity

import (
	"bytes"
	"errors"
	"regexp"
	"testing"
)

func TestGenerateDeterministicHexID(t *testing.T) {
	id, err := generate(bytes.NewReader([]byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	}))
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	if id != "000102030405060708090a0b0c0d0e0f" {
		t.Fatalf("generate() = %q", id)
	}
}

func TestGeneratedIDsUse128BitsOfLowercaseHex(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9a-f]{32}$`)
	for name, generateID := range map[string]func() (string, error){
		"paper":   NewPaperID,
		"comment": NewCommentID,
	} {
		t.Run(name, func(t *testing.T) {
			seen := make(map[string]struct{}, 100)
			for i := 0; i < 100; i++ {
				id, err := generateID()
				if err != nil {
					t.Fatalf("generate id: %v", err)
				}
				if !pattern.MatchString(id) {
					t.Fatalf("generated id = %q, want 32 lowercase hex characters", id)
				}
				if _, exists := seen[id]; exists {
					t.Fatalf("duplicate generated id %q", id)
				}
				seen[id] = struct{}{}
			}
		})
	}
}

func TestResolve(t *testing.T) {
	ids := []string{"a81f32c991b7", "a81f99bb0011", "410cbe130000", "410cbe130000ff"}
	tests := []struct {
		name     string
		selector string
		want     string
		wantErr  error
	}{
		{name: "exact", selector: "a81f32c991b7", want: "a81f32c991b7"},
		{name: "exact beats longer shared prefix", selector: "410cbe130000", want: "410cbe130000"},
		{name: "unique prefix", selector: "410cbe130000f", want: "410cbe130000ff"},
		{name: "ambiguous prefix", selector: "a81f", wantErr: ErrAmbiguous},
		{name: "missing prefix", selector: "ffff", wantErr: ErrNotFound},
		{name: "empty selector", selector: "", wantErr: ErrInvalid},
		{name: "unsafe selector", selector: "../a81f", wantErr: ErrInvalid},
		{name: "uppercase selector", selector: "A81F", wantErr: ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.selector, ids)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Resolve() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}
