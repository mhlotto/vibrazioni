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
			Kickoff:     time.Date(2026, time.April, 15, 20, 0, 0, 0, time.UTC),
			HomeTeam:    "Arsenal",
			AwayTeam:    "Sporting CP",
			Competition: "UEFA Champions League",
		},
	}

	if err := writeUpcomingFixtures(&out, fixtures, 7); err != nil {
		t.Fatalf("writeUpcomingFixtures() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Arsenal vs Sporting CP") {
		t.Fatalf("output missing fixture teams: %q", got)
	}
	if !strings.Contains(got, "UEFA Champions League") {
		t.Fatalf("output missing competition: %q", got)
	}
}

func TestWriteUpcomingFixturesEmpty(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := writeUpcomingFixtures(&out, nil, 7); err != nil {
		t.Fatalf("writeUpcomingFixtures() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "No upcoming fixtures in the next 7 days.") {
		t.Fatalf("unexpected output: %q", got)
	}
}
