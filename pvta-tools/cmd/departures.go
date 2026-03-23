package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"pvta-tools/pkg/models"
)

var departuresCmd = &cobra.Command{
	Use:   "departures <stop-id-or-name>",
	Short: "Show upcoming departures for a stop",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := newContext()
		defer cancel()

		departures, err := newAggregator().Departures(ctx, args[0])
		if err != nil {
			return err
		}
		if opts.JSON {
			return printJSON(departures)
		}

		fmt.Printf("Departures At Stop: %s\n", departures.Board.StopName)
		if departures.Board.LastUpdated != "" {
			fmt.Printf("Last Updated: %s\n", departures.Board.LastUpdated)
		}
		fmt.Println()
		if len(departures.Board.Groups) == 0 {
			fmt.Println("No upcoming departures listed for this stop")
			return nil
		}
		for _, group := range departures.Board.Groups {
			fmt.Printf("Service: %s\n", group.RouteAndDirection)
			if len(group.Times) == 0 {
				fmt.Println("  Upcoming departures: none listed")
			} else {
				fmt.Printf("  Upcoming departures at this stop: %s\n", strings.Join(group.Times, ", "))
			}
			if enriched := findEnrichedDepartureGroup(departures.EnrichedGroups, group.RouteAndDirection); enriched != nil {
				if enriched.MatchedRouteId != 0 {
					fmt.Printf("  Matched route: %s - %s (RouteId %d)\n", enriched.MatchedRouteShortName, enriched.MatchedRouteLongName, enriched.MatchedRouteId)
				}
				if len(enriched.LiveVehicles) > 0 {
					fmt.Println("  Buses likely approaching this stop on this service:")
					printDepartureVehicles(enriched.LiveVehicles, true)
				} else if len(enriched.DirectionVehicles) > 0 {
					fmt.Println("  Live buses on this service direction:")
					printDepartureVehicles(enriched.DirectionVehicles, false)
				} else if len(enriched.RouteVehicles) > 0 {
					fmt.Println("  Live buses on the matched route:")
					printDepartureVehicles(enriched.RouteVehicles, false)
				}
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(departuresCmd)
}

func findEnrichedDepartureGroup(groups []models.EnrichedDepartureGroup, routeAndDirection string) *models.EnrichedDepartureGroup {
	for i := range groups {
		if groups[i].RouteAndDirection == routeAndDirection {
			return &groups[i]
		}
	}
	return nil
}

func printDepartureVehicles(vehicles []models.ApproachingVehicle, includeStopsAway bool) {
	for _, vehicle := range vehicles {
		busLabel := strings.TrimSpace(vehicle.Name)
		if busLabel == "" {
			busLabel = strconv.Itoa(vehicle.VehicleId)
		}
		fmt.Printf("    - Bus %s", busLabel)
		if includeStopsAway && vehicle.StopsAway >= 0 {
			if vehicle.StopsAway == 0 {
				fmt.Printf(" is at this stop")
			} else {
				fmt.Printf(" is about %d stops away", vehicle.StopsAway)
			}
		}
		if vehicle.DirectionLong != "" {
			fmt.Printf(" [%s]", vehicle.DirectionLong)
		} else if vehicle.Direction != "" {
			fmt.Printf(" [%s]", vehicle.Direction)
		}
		if vehicle.DistanceMiles > 0 {
			fmt.Printf(" %.1f mi away", vehicle.DistanceMiles)
		}
		fmt.Println()
		if vehicle.CurrentStop != "" {
			fmt.Printf("      Current stop: %s\n", vehicle.CurrentStop)
		} else if vehicle.LastStop != "" {
			fmt.Printf("      Last stop: %s\n", vehicle.LastStop)
		}
		fmt.Printf("      Status: %s", vehicle.DisplayStatus)
		if vehicle.Deviation != 0 {
			fmt.Printf(" (%d min)", vehicle.Deviation)
		}
		fmt.Println()
	}
}
