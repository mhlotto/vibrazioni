package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/afc-tools/pkg/models"
)

func TestWritePastFixtures(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	fixtures := []models.Match{
		{
			ID:           600002,
			Kickoff:      time.Date(2026, time.April, 11, 11, 30, 0, 0, time.UTC),
			HomeTeam:     "Arsenal",
			AwayTeam:     "Bournemouth",
			RawScoreText: "1 - 2",
			Competition:  "Premier League",
		},
	}

	if err := writePastFixtures(&out, fixtures, 7, false); err != nil {
		t.Fatalf("writePastFixtures() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Arsenal 1 - 2 Bournemouth") {
		t.Fatalf("output missing scoreline: %q", got)
	}
	if !strings.Contains(got, "Premier League") {
		t.Fatalf("output missing competition: %q", got)
	}
	if strings.Contains(got, "600002 |") {
		t.Fatalf("unexpected id in non-football-data output: %q", got)
	}
}

func TestWritePastFixturesEmpty(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := writePastFixtures(&out, nil, 7, false); err != nil {
		t.Fatalf("writePastFixtures() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "No past fixtures in the last 7 days.") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestWritePastFixturesWithIDs(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	fixtures := []models.Match{
		{
			ID:           600002,
			Kickoff:      time.Date(2026, time.April, 11, 11, 30, 0, 0, time.UTC),
			HomeTeam:     "Arsenal FC",
			AwayTeam:     "AFC Bournemouth",
			RawScoreText: "1 - 2",
			Competition:  "Premier League",
		},
	}

	if err := writePastFixtures(&out, fixtures, 7, true); err != nil {
		t.Fatalf("writePastFixtures() error = %v", err)
	}

	if !strings.Contains(out.String(), "600002 |") {
		t.Fatalf("output missing id prefix: %q", out.String())
	}
}
