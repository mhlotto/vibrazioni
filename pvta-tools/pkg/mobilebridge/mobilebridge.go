package mobilebridge

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/app"
	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/client"
	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/models"
	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/service"
)

type Bridge struct {
	baseURL string
	timeout time.Duration
}

func NewBridge() *Bridge {
	return &Bridge{
		timeout: 10 * time.Second,
	}
}

func (b *Bridge) SetBaseURL(baseURL string) {
	b.baseURL = strings.TrimSpace(baseURL)
}

func (b *Bridge) BaseURL() string {
	return b.baseURL
}

func (b *Bridge) SetTimeoutSeconds(seconds int) {
	if seconds <= 0 {
		return
	}
	b.timeout = time.Duration(seconds) * time.Second
}

func (b *Bridge) RoutesJSON() (string, error) {
	ctx, cancel := b.newContext()
	defer cancel()

	routes, err := b.newAggregator().VisibleRoutes(ctx)
	if err != nil {
		return "", err
	}
	return marshalJSON(routes)
}

func (b *Bridge) VehiclesJSON() (string, error) {
	ctx, cancel := b.newContext()
	defer cancel()

	vehicles, err := b.newAggregator().VehiclePresentations(ctx)
	if err != nil {
		return "", err
	}
	return marshalJSON(vehicles)
}

func (b *Bridge) StopsJSON(filter string) (string, error) {
	ctx, cancel := b.newContext()
	defer cancel()

	stops, err := b.newAggregator().Stops(ctx)
	if err != nil {
		return "", err
	}
	filter = strings.TrimSpace(filter)
	if filter != "" {
		stops = filterStops(stops, filter)
	}
	return marshalJSON(stops)
}

func (b *Bridge) RouteStatusJSON(input string) (string, error) {
	ctx, cancel := b.newContext()
	defer cancel()

	status, err := b.newAggregator().RouteStatus(ctx, input)
	if err != nil {
		return "", err
	}
	return marshalJSON(status)
}

func (b *Bridge) StopStatusJSON(input string) (string, error) {
	ctx, cancel := b.newContext()
	defer cancel()

	status, err := b.newAggregator().StopStatus(ctx, input)
	if err != nil {
		return "", err
	}
	return marshalJSON(status)
}

func (b *Bridge) DeparturesJSON(input string) (string, error) {
	ctx, cancel := b.newContext()
	defer cancel()

	departures, err := b.newAggregator().Departures(ctx, input)
	if err != nil {
		return "", err
	}
	return marshalJSON(departures)
}

func (b *Bridge) RouteStopsJSON(input string) (string, error) {
	ctx, cancel := b.newContext()
	defer cancel()

	routeStops, err := b.newAggregator().RouteStops(ctx, input)
	if err != nil {
		return "", err
	}
	return marshalJSON(routeStops)
}

func (b *Bridge) newAggregator() *app.Aggregator {
	c := client.New(b.baseURL)
	return app.NewAggregator(
		service.NewRouteService(c),
		service.NewVehicleService(c),
		service.NewStopService(c),
		service.NewDepartureService(c),
	)
}

func (b *Bridge) newContext() (context.Context, context.CancelFunc) {
	timeout := b.timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

func marshalJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func filterStops(stops []models.Stop, filter string) []models.Stop {
	filter = strings.ToLower(filter)
	out := make([]models.Stop, 0, len(stops))
	for _, stop := range stops {
		if strings.Contains(strings.ToLower(stop.Name), filter) {
			out = append(out, stop)
		}
	}
	return out
}
