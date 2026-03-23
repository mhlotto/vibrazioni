package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"pvta-tools/pkg/models"
)

var stopsFilter string

var stopsCmd = &cobra.Command{
	Use:   "stops",
	Short: "List stops from GetAllStops",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := newContext()
		defer cancel()

		stops, err := newAggregator().Stops(ctx)
		if err != nil {
			return err
		}

		if stopsFilter != "" {
			filtered := make([]models.Stop, 0)
			lower := strings.ToLower(stopsFilter)
			for _, stop := range stops {
				if strings.Contains(strings.ToLower(stop.Name), lower) {
					filtered = append(filtered, stop)
				}
			}
			if opts.JSON {
				return printJSON(filtered)
			}
			for _, stop := range filtered {
				fmt.Printf("%d | %s\n", stop.StopId, stop.Name)
			}
			return nil
		}

		if opts.JSON {
			return printJSON(stops)
		}
		for _, stop := range stops {
			if stopsFilter != "" && !strings.Contains(strings.ToLower(stop.Name), strings.ToLower(stopsFilter)) {
				continue
			}
			fmt.Printf("%d | %s\n", stop.StopId, stop.Name)
		}
		return nil
	},
}

func init() {
	stopsCmd.Flags().StringVar(&stopsFilter, "filter", "", "Filter stops by case-insensitive name contains")
	rootCmd.AddCommand(stopsCmd)
}
