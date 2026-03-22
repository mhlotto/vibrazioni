package cmd

import "github.com/spf13/cobra"

var vehiclesCmd = &cobra.Command{
	Use:   "vehicles",
	Short: "List all live vehicles",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := newContext()
		defer cancel()

		vehicles, err := newAggregator().VehiclePresentations(ctx)
		if err != nil {
			return err
		}
		if opts.JSON {
			return printJSON(vehicles)
		}
		printVehicles(vehicles)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(vehiclesCmd)
}
