package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/mhlotto/vibrazioni/afc-tools/pkg/cache"
	"github.com/mhlotto/vibrazioni/afc-tools/pkg/client"
	"github.com/mhlotto/vibrazioni/afc-tools/pkg/models"
	"github.com/spf13/cobra"
)

var (
	pastDays     int
	pastCacheDir string
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

	rootCmd.AddCommand(pastCmd)
}

func runPast(w io.Writer) error {
	dataCache := cache.New(pastCacheDir)
	c := client.New("")

	fixtures, err := c.GetPastFixturesWithinNDays(context.Background(), pastDays, dataCache)
	if err != nil {
		return err
	}

	return writePastFixtures(w, fixtures, pastDays)
}

func writePastFixtures(w io.Writer, fixtures []models.Match, days int) error {
	if len(fixtures) == 0 {
		_, err := fmt.Fprintf(w, "No past fixtures in the last %d days.\n", days)
		return err
	}

	for _, fixture := range fixtures {
		kickoff := fixture.Kickoff.In(time.Local).Format("Mon Jan 2 15:04 MST")
		score := fixture.RawScoreText
		if _, err := fmt.Fprintf(
			w,
			"%s | %s %s %s | %s\n",
			kickoff,
			fixture.HomeTeam,
			score,
			fixture.AwayTeam,
			fixture.Competition,
		); err != nil {
			return err
		}
	}

	return nil
}
