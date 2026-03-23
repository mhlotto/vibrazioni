package service

import (
	"context"
	"fmt"

	"pvta-tools/pkg/client"
	"pvta-tools/pkg/models"
)

type VehicleService struct {
	client *client.Client
}

func NewVehicleService(c *client.Client) *VehicleService {
	return &VehicleService{client: c}
}

func (s *VehicleService) GetAllVehicles(ctx context.Context) ([]models.Vehicle, error) {
	var vehicles []models.Vehicle
	if err := s.client.GetJSON(ctx, "/Vehicles/GetAllVehicles", &vehicles); err != nil {
		return nil, err
	}
	return vehicles, nil
}

func (s *VehicleService) GetVehiclesForRoute(ctx context.Context, routeID int) ([]models.Vehicle, error) {
	var vehicles []models.Vehicle
	if err := s.client.GetJSON(ctx, fmt.Sprintf("/Vehicles/GetAllVehiclesForRoute?routeID=%d", routeID), &vehicles); err == nil {
		return vehicles, nil
	}
	vehicles, err := s.GetAllVehicles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.Vehicle, 0)
	for _, vehicle := range vehicles {
		if vehicle.RouteId == routeID {
			out = append(out, vehicle)
		}
	}
	return out, nil
}

func (s *VehicleService) GetVehiclesForStop(ctx context.Context, stopID int) ([]models.Vehicle, error) {
	vehicles, err := s.GetAllVehicles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.Vehicle, 0)
	for _, vehicle := range vehicles {
		if vehicle.StopId == stopID {
			out = append(out, vehicle)
		}
	}
	return out, nil
}
