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
	upcomingDays     int
	upcomingCacheDir string
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

	rootCmd.AddCommand(upcomingCmd)
}

func runUpcoming(w io.Writer) error {
	dataCache := cache.New(upcomingCacheDir)
	c := client.New("")

	fixtures, err := c.GetFixturesWithinNDays(context.Background(), upcomingDays, dataCache)
	if err != nil {
		return err
	}

	return writeUpcomingFixtures(w, fixtures, upcomingDays)
}

func writeUpcomingFixtures(w io.Writer, fixtures []models.Match, days int) error {
	if len(fixtures) == 0 {
		_, err := fmt.Fprintf(w, "No upcoming fixtures in the next %d days.\n", days)
		return err
	}

	for _, fixture := range fixtures {
		kickoff := fixture.Kickoff.In(time.Local).Format("Mon Jan 2 15:04 MST")
		if _, err := fmt.Fprintf(
			w,
			"%s | %s vs %s | %s\n",
			kickoff,
			fixture.HomeTeam,
			fixture.AwayTeam,
			fixture.Competition,
		); err != nil {
			return err
		}
	}

	return nil
}
