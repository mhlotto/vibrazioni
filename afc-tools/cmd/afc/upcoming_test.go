package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/afc-tools/pkg/models"
)

func TestWriteUpcomingFixtures(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	fixtures := []models.Match{
		{
			ID:          600001,
			Kickoff:     time.Date(2026, time.April, 15, 20, 0, 0, 0, time.UTC),
			HomeTeam:    "Arsenal",
			AwayTeam:    "Sporting CP",
			Competition: "UEFA Champions League",
		},
	}

	if err := writeUpcomingFixtures(&out, fixtures, 7, false); err != nil {
		t.Fatalf("writeUpcomingFixtures() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Arsenal vs Sporting CP") {
		t.Fatalf("output missing fixture teams: %q", got)
	}
	if !strings.Contains(got, "UEFA Champions League") {
		t.Fatalf("output missing competition: %q", got)
	}
	if strings.Contains(got, "600001 |") {
		t.Fatalf("unexpected id in non-football-data output: %q", got)
	}
}

func TestWriteUpcomingFixturesEmpty(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := writeUpcomingFixtures(&out, nil, 7, false); err != nil {
		t.Fatalf("writeUpcomingFixtures() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "No upcoming fixtures in the next 7 days.") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestWriteUpcomingFixturesWithIDs(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	fixtures := []models.Match{
		{
			ID:          600001,
			Kickoff:     time.Date(2026, time.April, 15, 20, 0, 0, 0, time.UTC),
			HomeTeam:    "Arsenal FC",
			AwayTeam:    "Sporting Clube de Portugal",
			Competition: "UEFA Champions League",
		},
	}

	if err := writeUpcomingFixtures(&out, fixtures, 7, true); err != nil {
		t.Fatalf("writeUpcomingFixtures() error = %v", err)
	}

	if !strings.Contains(out.String(), "600001 |") {
		t.Fatalf("output missing id prefix: %q", out.String())
	}
}
