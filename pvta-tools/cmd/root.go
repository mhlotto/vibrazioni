package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/app"
	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/client"
	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/models"
	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/service"
)

type options struct {
	BaseURL string
	JSON    bool
}

var opts options

var rootCmd = &cobra.Command{
	Use:   "pvta-tools",
	Short: "PVTA BusTracker CLI",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&opts.BaseURL, "base-url", "", "Override the InfoPoint REST base URL")
	rootCmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "Print JSON output")
}

func newAggregator() *app.Aggregator {
	c := client.New(opts.BaseURL)
	return app.NewAggregator(
		service.NewRouteService(c),
		service.NewVehicleService(c),
		service.NewStopService(c),
		service.NewDepartureService(c),
	)
}

func newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func printVehicles(vehicles []app.VehiclePresentation) {
	if len(vehicles) == 0 {
		fmt.Println("No active vehicles")
		return
	}
	for _, vehicle := range vehicles {
		v := vehicle.Vehicle
		fmt.Printf("- %d (%s)\n", v.VehicleId, valueOrFallback(v.DirectionLong, "Unknown direction"))
		fmt.Printf("  Location: %.6f, %.6f\n", v.Latitude, v.Longitude)
		fmt.Printf("  Last Stop: %s\n", valueOrFallback(v.LastStop, "Unknown"))
		if vehicle.CurrentStop != nil {
			fmt.Printf("  Current Stop: %s\n", valueOrFallback(vehicle.CurrentStop.Name, "Unknown"))
		}
		if vehicle.NextStop != nil {
			fmt.Printf("  Next Stop: %s\n", valueOrFallback(vehicle.NextStop.Name, "Unknown"))
		}
		fmt.Printf("  Status: %s\n", valueOrFallback(v.DisplayStatus, "Unknown"))
		fmt.Printf("  Deviation: %d min\n", v.Deviation)
		fmt.Printf("  Occupancy: %s\n", valueOrFallback(v.OccupancyStatusReportLabel, "Unknown"))
	}
}

func printMessages(messages []models.Message) {
	if len(messages) == 0 {
		fmt.Println("Alerts: none")
		return
	}
	fmt.Println("Alerts:")
	for _, message := range messages {
		fmt.Printf("- %s\n", valueOrFallback(app.FormatMessageSummary(message), "Alert"))
	}
}

func valueOrFallback(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
