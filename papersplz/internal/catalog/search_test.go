package catalog

import (
	"reflect"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

func TestSearchPapersManagedFieldsAndANDTerms(t *testing.T) {
	catalogPath := newTestCatalog(t)
	timestamp := time.Date(2026, 8, 24, 16, 0, 0, 0, time.UTC)
	papers := []model.Paper{
		testInspectionPaper("aaaa00000000", "Spectral Methods", []string{"Alice Smith"}, []string{"topology"}),
		testInspectionPaper("bbbb00000000", "Gamma", []string{"Bob Jones"}, []string{"algebra"}),
		testInspectionPaper("cccc00000000", "Delta", []string{"Carol Example"}, []string{"geometry"}),
	}
	papers[0].Source = "Annals of Examples"
	papers[0].Review = &model.Review{Text: "Uses a Serre sequence", CreatedAt: timestamp, UpdatedAt: timestamp}
	papers[0].Comments = []model.Comment{{ID: "11110000", Text: "Check Lemma Four", CreatedAt: timestamp, UpdatedAt: timestamp}}
	for _, paper := range papers {
		writeCatalogPaper(t, catalogPath, paper)
	}

	tests := []struct {
		name    string
		options SearchOptions
		want    []string
	}{
		{name: "title case insensitive", options: SearchOptions{Terms: []string{"SPECTRAL"}}, want: []string{"Spectral Methods"}},
		{name: "author", options: SearchOptions{Terms: []string{"smith"}}, want: []string{"Spectral Methods"}},
		{name: "source", options: SearchOptions{Terms: []string{"annals"}}, want: []string{"Spectral Methods"}},
		{name: "tag", options: SearchOptions{Terms: []string{"topology"}}, want: []string{"Spectral Methods"}},
		{name: "review", options: SearchOptions{Terms: []string{"serre"}}, want: []string{"Spectral Methods"}},
		{name: "comment", options: SearchOptions{Terms: []string{"lemma four"}}, want: []string{"Spectral Methods"}},
		{name: "cross field AND", options: SearchOptions{Terms: []string{"spectral", "smith", "serre", "lemma"}}, want: []string{"Spectral Methods"}},
		{name: "non-match", options: SearchOptions{Terms: []string{"spectral", "missing"}}, want: []string{}},
		{name: "plain not regexp", options: SearchOptions{Terms: []string{"Spectral.*Methods"}}, want: []string{}},
		{name: "filters compose", options: SearchOptions{Terms: []string{"sequence"}, Tag: " TOPOLOGY ", Author: "ALICE"}, want: []string{"Spectral Methods"}},
		{name: "filter excludes", options: SearchOptions{Terms: []string{"sequence"}, Tag: "algebra"}, want: []string{}},
		{name: "empty terms list all", options: SearchOptions{}, want: []string{"Delta", "Gamma", "Spectral Methods"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SearchPapers(catalogPath, tt.options)
			if err != nil {
				t.Fatal(err)
			}
			titles := make([]string, len(got))
			for i := range got {
				titles[i] = got[i].Title
			}
			if !reflect.DeepEqual(titles, tt.want) {
				t.Fatalf("titles = %v, want %v", titles, tt.want)
			}
		})
	}
}
