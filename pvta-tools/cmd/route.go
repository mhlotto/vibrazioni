package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var routeCmd = &cobra.Command{
	Use:   "route <route>",
	Short: "Show route status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := newContext()
		defer cancel()

		status, err := newAggregator().RouteStatus(ctx, args[0])
		if err != nil {
			return err
		}
		if opts.JSON {
			return printJSON(status)
		}

		fmt.Printf("Route: %s - %s\n\n", status.Route.ShortName, status.Route.LongName)
		printMessages(status.Messages)
		fmt.Println()
		fmt.Println("Vehicles:")
		printVehicles(status.Vehicles)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(routeCmd)
}
