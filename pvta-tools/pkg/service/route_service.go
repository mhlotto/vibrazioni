package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"pvta-tools/pkg/client"
	"pvta-tools/pkg/models"
)

type RouteService struct {
	client *client.Client
}

func NewRouteService(c *client.Client) *RouteService {
	return &RouteService{client: c}
}

func (s *RouteService) GetAllRoutes(ctx context.Context) ([]models.Route, error) {
	var routes []models.Route
	if err := s.client.GetJSON(ctx, "/Routes/GetAllRoutes", &routes); err != nil {
		return nil, err
	}
	return routes, nil
}

func (s *RouteService) GetVisibleRoutes(ctx context.Context) ([]models.Route, error) {
	var routes []models.Route
	if err := s.client.GetJSON(ctx, "/Routes/GetVisibleRoutes", &routes); err != nil {
		return nil, err
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].SortOrder == routes[j].SortOrder {
			return routes[i].ShortName < routes[j].ShortName
		}
		return routes[i].SortOrder < routes[j].SortOrder
	})
	return routes, nil
}

func (s *RouteService) VisibleRoutes(ctx context.Context) ([]models.Route, error) {
	routes, err := s.GetVisibleRoutes(ctx)
	if err != nil {
		return nil, err
	}
	return routes, nil
}

func (s *RouteService) FindRoute(ctx context.Context, input string) (*models.Route, error) {
	routes, err := s.GetVisibleRoutes(ctx)
	if err != nil {
		return nil, err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("route input is required")
	}
	if id, err := strconv.Atoi(input); err == nil {
		for _, route := range routes {
			if route.RouteId == id {
				routeCopy := route
				return &routeCopy, nil
			}
		}
	}
	for _, route := range routes {
		if strings.EqualFold(route.ShortName, input) {
			routeCopy := route
			return &routeCopy, nil
		}
	}
	return nil, fmt.Errorf("route not found: %s", input)
}

func (s *RouteService) GetRouteDetails(ctx context.Context, routeID int) (*models.RouteDetail, error) {
	var detail models.RouteDetail
	if err := s.client.GetJSON(ctx, fmt.Sprintf("/RouteDetails/Get/%d", routeID), &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}
