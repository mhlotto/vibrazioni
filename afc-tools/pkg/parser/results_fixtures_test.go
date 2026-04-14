package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mhlotto/vibrazioni/afc-tools/pkg/models"
)

func TestParseResultsAndFixturesList(t *testing.T) {
	t.Parallel()

	base := filepath.Join("..", "client", "testdata")

	input, err := os.ReadFile(filepath.Join(base, "results-and-fixtures-get.out"))
	if err != nil {
		t.Fatalf("ReadFile(input) error = %v", err)
	}

	got, err := ParseResultsAndFixturesList(input)
	if err != nil {
		t.Fatalf("ParseResultsAndFixturesList() error = %v", err)
	}

	expectedBytes, err := os.ReadFile(filepath.Join(base, "results-and-fixtures.json"))
	if err != nil {
		t.Fatalf("ReadFile(expected) error = %v", err)
	}

	var want models.ResultsAndFixturesList
	if err := json.Unmarshal(expectedBytes, &want); err != nil {
		t.Fatalf("json.Unmarshal(expected) error = %v", err)
	}

	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("ParseResultsAndFixturesList() mismatch\n got: %#v\nwant: %#v", *got, want)
	}
}
