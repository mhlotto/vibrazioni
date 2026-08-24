package model

import (
	"errors"
	"testing"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
)

func TestResolveCommentID(t *testing.T) {
	paper := Paper{Comments: []Comment{
		{ID: "c91e84f20000"},
		{ID: "c91effff1111"},
		{ID: "410cbe130000"},
	}}
	tests := []struct {
		name     string
		selector string
		want     string
		wantErr  error
	}{
		{name: "exact", selector: "c91e84f20000", want: "c91e84f20000"},
		{name: "unique prefix", selector: "410c", want: "410cbe130000"},
		{name: "ambiguous", selector: "c91e", wantErr: identity.ErrAmbiguous},
		{name: "missing", selector: "ffff", wantErr: identity.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveCommentID(paper, tt.selector)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolveCommentID() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ResolveCommentID() = %q, want %q", got, tt.want)
			}
		})
	}
}
