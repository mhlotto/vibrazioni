package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/mhlotto/vibrazioni/afc-tools/pkg/cache"
	"github.com/mhlotto/vibrazioni/afc-tools/pkg/models"
	"github.com/spf13/cobra"
)

var (
	pastDays     int
	pastCacheDir string
	pastSource   string
)

func init() {
	defaultCacheDir, err := cache.DefaultDir()
	if err != nil {
		defaultCacheDir = ".afctoolcache"
	}

	pastCmd := &cobra.Command{
		Use:   "past",
		Short: "Show recent Arsenal results",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPast(cmd.OutOrStdout())
		},
	}

	pastCmd.Flags().IntVar(&pastDays, "days", 7, "Number of days back to include")
	pastCmd.Flags().StringVar(&pastCacheDir, "cache-dir", defaultCacheDir, "Cache directory path")
	pastCmd.Flags().StringVar(&pastSource, "source", sourceArsenal, "Data source: arsenal or football-data")

	rootCmd.AddCommand(pastCmd)
}

func runPast(w io.Writer) error {
	fixtures, err := pastFixtures(context.Background(), pastSource, pastDays, pastCacheDir)
	if err != nil {
		return err
	}

	return writePastFixtures(w, fixtures, pastDays, normalizedSource(pastSource) == sourceFootballData)
}

func writePastFixtures(w io.Writer, fixtures []models.Match, days int, showIDs bool) error {
	if len(fixtures) == 0 {
		_, err := fmt.Fprintf(w, "No past fixtures in the last %d days.\n", days)
		return err
	}

	for _, fixture := range fixtures {
		kickoff := fixture.Kickoff.In(time.Local).Format("Mon Jan 2 15:04 MST")
		score := fixture.RawScoreText
		line := fmt.Sprintf(
			"%s | %s %s %s | %s",
			kickoff,
			fixture.HomeTeam,
			score,
			fixture.AwayTeam,
			fixture.Competition,
		)
		if showIDs && fixture.ID > 0 {
			line = fmt.Sprintf("%d | %s", fixture.ID, line)
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	return nil
}
