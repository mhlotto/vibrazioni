package service

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/client"
	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/models"
)

type StopService struct {
	client *client.Client
}

func NewStopService(c *client.Client) *StopService {
	return &StopService{client: c}
}

func (s *StopService) GetAllStops(ctx context.Context) ([]models.Stop, error) {
	var stops []models.Stop
	if err := s.client.GetJSON(ctx, "/Stops/GetAllStops", &stops); err != nil {
		return nil, err
	}
	return stops, nil
}

func (s *StopService) GetAllStopsForRoute(ctx context.Context, routeID int) ([]models.Stop, error) {
	var stops []models.Stop
	path := "/Stops/GetAllStopsForRoutes?routeIDs=" + url.QueryEscape(strconv.Itoa(routeID))
	if err := s.client.GetJSON(ctx, path, &stops); err != nil {
		return nil, err
	}
	return stops, nil
}

func (s *StopService) FindStop(ctx context.Context, input string) (*models.Stop, error) {
	stops, err := s.GetAllStops(ctx)
	if err != nil {
		return nil, err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("stop input is required")
	}
	if id, err := strconv.Atoi(input); err == nil {
		if stop, ok := bestStopByID(stops, id); ok {
			return &stop, nil
		}
	}
	lowerInput := strings.ToLower(input)
	for _, stop := range stops {
		if strings.Contains(strings.ToLower(stop.Name), lowerInput) {
			stopCopy := stop
			return &stopCopy, nil
		}
	}
	return nil, fmt.Errorf("stop not found: %s", input)
}

func bestStopByID(stops []models.Stop, stopID int) (models.Stop, bool) {
	var (
		best  models.Stop
		found bool
	)
	for _, stop := range stops {
		if stop.StopId != stopID {
			continue
		}
		if !found || (!best.IsTimePoint && stop.IsTimePoint) || (best.Name == "" && stop.Name != "") {
			best = stop
			found = true
		}
	}
	return best, found
}
