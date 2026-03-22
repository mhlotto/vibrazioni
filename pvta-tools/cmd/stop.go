package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <stop-id-or-name>",
	Short: "Lookup stop status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := newContext()
		defer cancel()

		status, err := newAggregator().StopStatus(ctx, args[0])
		if err != nil {
			return err
		}
		if opts.JSON {
			return printJSON(status)
		}

		fmt.Printf("Stop: %s\n\n", status.Stop.Name)
		if len(status.Vehicles) == 0 {
			fmt.Println("Live stop predictions not available yet")
			return nil
		}
		fmt.Println("Vehicles near/at stop:")
		printVehicles(status.Vehicles)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
