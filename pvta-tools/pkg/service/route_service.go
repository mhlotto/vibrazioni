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

func (s *RouteService) VisibleRoutes(ctx context.Context) ([]models.Route, error) {
	routes, err := s.GetAllRoutes(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]models.Route, 0, len(routes))
	for _, route := range routes {
		if route.IsVisible {
			filtered = append(filtered, route)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].SortOrder == filtered[j].SortOrder {
			return filtered[i].ShortName < filtered[j].ShortName
		}
		return filtered[i].SortOrder < filtered[j].SortOrder
	})
	return filtered, nil
}

func (s *RouteService) FindRoute(ctx context.Context, input string) (*models.Route, error) {
	routes, err := s.GetAllRoutes(ctx)
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
