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
	upcomingDays     int
	upcomingCacheDir string
	upcomingSource   string
)

func init() {
	defaultCacheDir, err := cache.DefaultDir()
	if err != nil {
		defaultCacheDir = ".afctoolcache"
	}

	upcomingCmd := &cobra.Command{
		Use:   "upcoming",
		Short: "Show upcoming Arsenal fixtures",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpcoming(cmd.OutOrStdout())
		},
	}

	upcomingCmd.Flags().IntVar(&upcomingDays, "days", 7, "Number of days ahead to include")
	upcomingCmd.Flags().StringVar(&upcomingCacheDir, "cache-dir", defaultCacheDir, "Cache directory path")
	upcomingCmd.Flags().StringVar(&upcomingSource, "source", sourceArsenal, "Data source: arsenal or football-data")

	rootCmd.AddCommand(upcomingCmd)
}

func runUpcoming(w io.Writer) error {
	fixtures, err := upcomingFixtures(context.Background(), upcomingSource, upcomingDays, upcomingCacheDir)
	if err != nil {
		return err
	}

	return writeUpcomingFixtures(w, fixtures, upcomingDays, normalizedSource(upcomingSource) == sourceFootballData)
}

func writeUpcomingFixtures(w io.Writer, fixtures []models.Match, days int, showIDs bool) error {
	if len(fixtures) == 0 {
		_, err := fmt.Fprintf(w, "No upcoming fixtures in the next %d days.\n", days)
		return err
	}

	for _, fixture := range fixtures {
		kickoff := fixture.Kickoff.In(time.Local).Format("Mon Jan 2 15:04 MST")
		line := fmt.Sprintf(
			"%s | %s vs %s | %s",
			kickoff,
			fixture.HomeTeam,
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
