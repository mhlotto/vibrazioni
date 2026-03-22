package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "List visible routes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := newContext()
		defer cancel()

		routes, err := newAggregator().VisibleRoutes(ctx)
		if err != nil {
			return err
		}
		if opts.JSON {
			return printJSON(routes)
		}
		for _, route := range routes {
			fmt.Printf("%s | %s | %d\n", route.ShortName, route.LongName, route.RouteId)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(routesCmd)
}
