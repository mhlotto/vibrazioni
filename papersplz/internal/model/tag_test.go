package model

import (
	"reflect"
	"testing"
)

func TestNormalizeTag(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "  Algebraic-Topology  ", want: "algebraic-topology"},
		{input: "category_theory", want: "category_theory"},
		{input: "p-adic", want: "p-adic"},
		{input: "qft+notes.v1", want: "qft+notes.v1"},
		{input: "has spaces", wantErr: true},
		{input: "-leading", wantErr: true},
		{input: "", wantErr: true},
		{input: "café", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := NormalizeTag(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeTag() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("NormalizeTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeTagsDeduplicatesInOrder(t *testing.T) {
	got, err := NormalizeTags([]string{" Math ", "to-read", "MATH", "qft"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"math", "to-read", "qft"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeTags() = %v, want %v", got, want)
	}
}
