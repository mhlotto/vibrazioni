package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var routeStopsCmd = &cobra.Command{
	Use:   "route-stops <route>",
	Short: "List ordered stops for a route",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := newContext()
		defer cancel()

		status, err := newAggregator().RouteStops(ctx, args[0])
		if err != nil {
			return err
		}
		if opts.JSON {
			return printJSON(status)
		}

		fmt.Printf("Route: %s - %s\n\n", status.Route.ShortName, status.Route.LongName)
		for _, stop := range status.Stops {
			fmt.Printf("%d | %s\n", stop.StopId, stop.Name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(routeStopsCmd)
}
