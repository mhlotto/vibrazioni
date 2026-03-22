package app

import (
	"context"
	"fmt"
	"strings"

	"pvta-tools/pkg/models"
	"pvta-tools/pkg/service"
)

type Aggregator struct {
	routes   *service.RouteService
	vehicles *service.VehicleService
	stops    *service.StopService
}

type RouteStatus struct {
	Route    models.Route          `json:"route"`
	Vehicles []VehiclePresentation `json:"vehicles"`
	Messages []models.Message      `json:"messages"`
}

type StopStatus struct {
	Stop     models.Stop           `json:"stop"`
	Vehicles []VehiclePresentation `json:"vehicles"`
}

type VehiclePresentation struct {
	Vehicle     models.Vehicle `json:"vehicle"`
	CurrentStop *models.Stop   `json:"current_stop,omitempty"`
	NextStop    *models.Stop   `json:"next_stop,omitempty"`
}

func NewAggregator(routeSvc *service.RouteService, vehicleSvc *service.VehicleService, stopSvc *service.StopService) *Aggregator {
	return &Aggregator{
		routes:   routeSvc,
		vehicles: vehicleSvc,
		stops:    stopSvc,
	}
}

func (a *Aggregator) VisibleRoutes(ctx context.Context) ([]models.Route, error) {
	return a.routes.VisibleRoutes(ctx)
}

func (a *Aggregator) Vehicles(ctx context.Context) ([]models.Vehicle, error) {
	return a.vehicles.GetAllVehicles(ctx)
}

func (a *Aggregator) VehiclePresentations(ctx context.Context) ([]VehiclePresentation, error) {
	vehicles, err := a.vehicles.GetAllVehicles(ctx)
	if err != nil {
		return nil, err
	}
	return a.presentVehicles(ctx, vehicles)
}

func (a *Aggregator) RouteStatus(ctx context.Context, input string) (*RouteStatus, error) {
	route, err := a.routes.FindRoute(ctx, input)
	if err != nil {
		return nil, err
	}
	vehicles, err := a.vehicles.GetVehiclesForRoute(ctx, route.RouteId)
	if err != nil {
		return nil, err
	}
	presented, err := a.presentVehicles(ctx, vehicles)
	if err != nil {
		return nil, err
	}
	return &RouteStatus{
		Route:    *route,
		Vehicles: presented,
		Messages: route.Messages,
	}, nil
}

func (a *Aggregator) StopStatus(ctx context.Context, input string) (*StopStatus, error) {
	stop, err := a.stops.FindStop(ctx, input)
	if err != nil {
		return nil, err
	}
	vehicles, err := a.vehicles.GetVehiclesForStop(ctx, stop.StopId)
	if err != nil {
		return nil, err
	}
	presented, err := a.presentVehicles(ctx, vehicles)
	if err != nil {
		return nil, err
	}
	return &StopStatus{
		Stop:     *stop,
		Vehicles: presented,
	}, nil
}

func (a *Aggregator) presentVehicles(ctx context.Context, vehicles []models.Vehicle) ([]VehiclePresentation, error) {
	routeStopsCache := make(map[int][]models.Stop)
	out := make([]VehiclePresentation, 0, len(vehicles))
	for _, vehicle := range vehicles {
		p := VehiclePresentation{Vehicle: vehicle}
		if vehicle.RouteId != 0 && vehicle.StopId != 0 {
			stops, ok := routeStopsCache[vehicle.RouteId]
			if !ok {
				var err error
				stops, err = a.stops.GetAllStopsForRoute(ctx, vehicle.RouteId)
				if err != nil {
					return nil, fmt.Errorf("load stops for route %d: %w", vehicle.RouteId, err)
				}
				routeStopsCache[vehicle.RouteId] = stops
			}
			p.CurrentStop, p.NextStop = findCurrentAndNextStop(stops, vehicle.StopId)
		}
		out = append(out, p)
	}
	return out, nil
}

func findCurrentAndNextStop(stops []models.Stop, stopID int) (*models.Stop, *models.Stop) {
	for i, stop := range stops {
		if stop.StopId != stopID {
			continue
		}
		current := stop
		var next *models.Stop
		for j := i + 1; j < len(stops); j++ {
			if stops[j].StopId == stopID {
				continue
			}
			candidate := stops[j]
			next = &candidate
			break
		}
		return &current, next
	}
	return nil, nil
}

func FormatMessageSummary(message models.Message) string {
	text := strings.TrimSpace(message.Header)
	if text == "" {
		text = strings.TrimSpace(message.Message)
	}
	return text
}
