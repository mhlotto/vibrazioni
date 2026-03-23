package app

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"pvta-tools/pkg/models"
	"pvta-tools/pkg/service"
)

type Aggregator struct {
	routes     *service.RouteService
	vehicles   *service.VehicleService
	stops      *service.StopService
	departures *service.DepartureService
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

func NewAggregator(routeSvc *service.RouteService, vehicleSvc *service.VehicleService, stopSvc *service.StopService, departureSvc *service.DepartureService) *Aggregator {
	return &Aggregator{
		routes:     routeSvc,
		vehicles:   vehicleSvc,
		stops:      stopSvc,
		departures: departureSvc,
	}
}

func (a *Aggregator) VisibleRoutes(ctx context.Context) ([]models.Route, error) {
	return a.routes.VisibleRoutes(ctx)
}

func (a *Aggregator) Vehicles(ctx context.Context) ([]models.Vehicle, error) {
	return a.vehicles.GetAllVehicles(ctx)
}

func (a *Aggregator) Stops(ctx context.Context) ([]models.Stop, error) {
	stops, err := a.stops.GetAllStops(ctx)
	if err != nil {
		return nil, err
	}
	stops = dedupeStops(stops)
	sort.Slice(stops, func(i, j int) bool {
		if stops[i].Name == stops[j].Name {
			if stops[i].StopId == stops[j].StopId {
				return stops[i].StopRecordId < stops[j].StopRecordId
			}
			return stops[i].StopId < stops[j].StopId
		}
		return stops[i].Name < stops[j].Name
	})
	return stops, nil
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
	presented, err := a.presentVehicles(ctx, route.Vehicles)
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

func (a *Aggregator) RouteStops(ctx context.Context, input string) (*RouteStops, error) {
	route, err := a.routes.FindRoute(ctx, input)
	if err != nil {
		return nil, err
	}
	stops, err := a.stops.GetAllStopsForRoute(ctx, route.RouteId)
	if err != nil {
		return nil, err
	}
	return &RouteStops{
		Route: *route,
		Stops: stops,
	}, nil
}

func (a *Aggregator) Departures(ctx context.Context, input string) (*StopDepartures, error) {
	stop, err := a.stops.FindStop(ctx, input)
	if err != nil {
		return nil, err
	}
	board, err := a.departures.GetBoardForStop(ctx, stop.StopId)
	if err != nil {
		return nil, err
	}
	enriched, err := a.enrichDepartureGroups(ctx, stop.StopId, board.Groups)
	if err != nil {
		return nil, err
	}
	return &StopDepartures{
		Stop:           *stop,
		Board:          *board,
		EnrichedGroups: enriched,
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

func dedupeStops(stops []models.Stop) []models.Stop {
	byID := make(map[int]models.Stop)
	order := make([]int, 0, len(stops))
	for _, stop := range stops {
		current, ok := byID[stop.StopId]
		if !ok {
			byID[stop.StopId] = stop
			order = append(order, stop.StopId)
			continue
		}
		if (!current.IsTimePoint && stop.IsTimePoint) || (current.Name == "" && stop.Name != "") {
			byID[stop.StopId] = stop
		}
	}
	out := make([]models.Stop, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func (a *Aggregator) enrichDepartureGroups(ctx context.Context, stopID int, groups []models.DepartureGroup) ([]models.EnrichedDepartureGroup, error) {
	routes, err := a.routes.GetVisibleRoutes(ctx)
	if err != nil {
		return nil, err
	}
	routeIndex := make(map[string]models.Route, len(routes)*3)
	for _, route := range routes {
		routeIndex[normalizeRouteLabel(route.LongName)] = route
		routeIndex[normalizeRouteLabel(route.ShortName)] = route
		routeIndex[normalizeRouteLabel(route.RouteAbbreviation)] = route
	}

	stops, err := a.Stops(ctx)
	if err != nil {
		return nil, err
	}
	stopMap := make(map[int]models.Stop, len(stops))
	for _, stop := range stops {
		stopMap[stop.StopId] = stop
	}

	detailCache := make(map[int]*models.RouteDetail)
	vehiclesCache := make(map[int][]models.Vehicle)
	orderedStopsCache := make(map[int][]models.Stop)

	out := make([]models.EnrichedDepartureGroup, 0, len(groups))
	for _, group := range groups {
		enriched := models.EnrichedDepartureGroup{
			RouteAndDirection: group.RouteAndDirection,
			Times:             group.Times,
		}
		routeName, directionLabel := splitRouteAndDirection(group.RouteAndDirection)
		if routeName == "" || directionLabel == "" {
			out = append(out, enriched)
			continue
		}
		route, ok := routeIndex[normalizeRouteLabel(routeName)]
		if !ok {
			out = append(out, enriched)
			continue
		}
		enriched.MatchedRouteId = route.RouteId
		enriched.MatchedRouteShortName = route.ShortName
		enriched.MatchedRouteLongName = route.LongName

		detail, ok := detailCache[route.RouteId]
		if !ok {
			detail, err = a.routes.GetRouteDetails(ctx, route.RouteId)
			if err != nil {
				return nil, err
			}
			detailCache[route.RouteId] = detail
		}

		vehicles, ok := vehiclesCache[route.RouteId]
		if !ok {
			vehicles, err = a.vehicles.GetVehiclesForRoute(ctx, route.RouteId)
			if err != nil {
				return nil, err
			}
			vehiclesCache[route.RouteId] = vehicles
		}
		orderedStops, ok := orderedStopsCache[route.RouteId]
		if !ok {
			orderedStops, err = a.stops.GetAllStopsForRoute(ctx, route.RouteId)
			if err != nil {
				return nil, err
			}
			orderedStopsCache[route.RouteId] = orderedStops
		}

		enriched.RouteVehicles = summarizeVehicles(vehicles, stopID, stopMap)
		enriched.DirectionVehicles = findDirectionVehicles(orderedStops, vehicles, stopID, directionLabel, stopMap)
		enriched.LiveVehicles = findApproachingVehicles(detail, vehicles, stopID, directionLabel, stopMap)
		if len(enriched.LiveVehicles) == 0 {
			enriched.StopInRouteSequence = routeContainsStop(orderedStops, stopID)
			enriched.LiveVehicles = findApproachingVehiclesFromOrderedStops(orderedStops, vehicles, stopID, directionLabel, stopMap)
		} else {
			enriched.StopInRouteSequence = stopInRouteDetail(detail, stopID)
		}
		out = append(out, enriched)
	}
	return out, nil
}

func splitRouteAndDirection(s string) (string, string) {
	s = strings.TrimSpace(s)
	idx := strings.LastIndex(s, " - ")
	if idx == -1 {
		return "", ""
	}
	return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+3:])
}

func findApproachingVehicles(detail *models.RouteDetail, vehicles []models.Vehicle, targetStopID int, directionLabel string, stopMap map[int]models.Stop) []models.ApproachingVehicle {
	dirCode := directionCodeFromLabel(directionLabel)
	if dirCode == "" {
		return nil
	}

	stopOrders := make(map[int]int)
	targetOrders := make([]int, 0)
	for _, routeStop := range detail.RouteStops {
		if !strings.EqualFold(routeStop.Direction, dirCode) {
			continue
		}
		stopOrders[routeStop.StopId] = routeStop.SortOrder
		if routeStop.StopId == targetStopID {
			targetOrders = append(targetOrders, routeStop.SortOrder)
		}
	}
	if len(targetOrders) == 0 {
		return nil
	}

	out := make([]models.ApproachingVehicle, 0)
	for _, vehicle := range vehicles {
		if !vehicleMatchesDirection(vehicle, directionLabel, dirCode) {
			continue
		}
		currentOrder, ok := stopOrders[vehicle.StopId]
		if !ok {
			continue
		}
		bestStopsAway := -1
		for _, targetOrder := range targetOrders {
			if currentOrder > targetOrder {
				continue
			}
			delta := targetOrder - currentOrder
			if bestStopsAway == -1 || delta < bestStopsAway {
				bestStopsAway = delta
			}
		}
		if bestStopsAway == -1 {
			continue
		}
		currentStop := ""
		if stop, ok := stopMap[vehicle.StopId]; ok {
			currentStop = stop.Name
		}
		out = append(out, models.ApproachingVehicle{
			VehicleId:                  vehicle.VehicleId,
			Name:                       vehicle.Name,
			Direction:                  vehicle.Direction,
			DirectionLong:              vehicle.DirectionLong,
			CurrentStop:                currentStop,
			LastStop:                   vehicle.LastStop,
			StopsAway:                  bestStopsAway,
			DistanceMiles:              distanceToTargetStop(vehicle, targetStopID, stopMap),
			Deviation:                  vehicle.Deviation,
			DisplayStatus:              vehicle.DisplayStatus,
			OccupancyStatusReportLabel: vehicle.OccupancyStatusReportLabel,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].StopsAway == out[j].StopsAway {
			if out[i].Deviation == out[j].Deviation {
				return out[i].VehicleId < out[j].VehicleId
			}
			return out[i].Deviation < out[j].Deviation
		}
		return out[i].StopsAway < out[j].StopsAway
	})
	return out
}

func directionCodeFromLabel(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "north", "northbound":
		return "N"
	case "south", "southbound":
		return "S"
	case "east", "eastbound":
		return "E"
	case "west", "westbound":
		return "W"
	case "upward":
		return "UP"
	case "downward":
		return "DWN"
	case "counter clock wise":
		return "CCW"
	case "clock wise":
		return "CW"
	default:
		return ""
	}
}

func vehicleMatchesDirection(vehicle models.Vehicle, directionLabel, dirCode string) bool {
	if strings.EqualFold(vehicle.DirectionLong, directionLabel) {
		return true
	}
	return strings.EqualFold(vehicle.Direction, dirCode)
}

func normalizeRouteLabel(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "/", " / ")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, "  ", " ")
	return s
}

func findApproachingVehiclesFromOrderedStops(orderedStops []models.Stop, vehicles []models.Vehicle, targetStopID int, directionLabel string, stopMap map[int]models.Stop) []models.ApproachingVehicle {
	dirCode := directionCodeFromLabel(directionLabel)
	if dirCode == "" {
		return nil
	}
	targetIndexes := make([]int, 0)
	for i, stop := range orderedStops {
		if stop.StopId == targetStopID {
			targetIndexes = append(targetIndexes, i)
		}
	}
	if len(targetIndexes) == 0 {
		return nil
	}

	out := make([]models.ApproachingVehicle, 0)
	for _, vehicle := range vehicles {
		currentIndexes := indexesForStopID(orderedStops, vehicle.StopId)
		if len(currentIndexes) == 0 {
			continue
		}

		bestStopsAway := -1
		for _, currentIdx := range currentIndexes {
			inferred := inferDirectionFromOrderedStops(orderedStops, currentIdx)
			if inferred != "" && inferred != dirCode {
				continue
			}
			for _, targetIdx := range targetIndexes {
				if currentIdx > targetIdx {
					continue
				}
				delta := targetIdx - currentIdx
				if bestStopsAway == -1 || delta < bestStopsAway {
					bestStopsAway = delta
				}
			}
		}
		if bestStopsAway == -1 {
			continue
		}

		currentStop := ""
		if stop, ok := stopMap[vehicle.StopId]; ok {
			currentStop = stop.Name
		}
		out = append(out, models.ApproachingVehicle{
			VehicleId:                  vehicle.VehicleId,
			Name:                       vehicle.Name,
			Direction:                  vehicle.Direction,
			DirectionLong:              vehicle.DirectionLong,
			CurrentStop:                currentStop,
			LastStop:                   vehicle.LastStop,
			StopsAway:                  bestStopsAway,
			DistanceMiles:              distanceToTargetStop(vehicle, targetStopID, stopMap),
			Deviation:                  vehicle.Deviation,
			DisplayStatus:              vehicle.DisplayStatus,
			OccupancyStatusReportLabel: vehicle.OccupancyStatusReportLabel,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].StopsAway == out[j].StopsAway {
			if out[i].Deviation == out[j].Deviation {
				return out[i].VehicleId < out[j].VehicleId
			}
			return out[i].Deviation < out[j].Deviation
		}
		return out[i].StopsAway < out[j].StopsAway
	})
	return out
}

func indexesForStopID(stops []models.Stop, stopID int) []int {
	out := make([]int, 0)
	for i, stop := range stops {
		if stop.StopId == stopID {
			out = append(out, i)
		}
	}
	return out
}

func inferDirectionFromOrderedStops(stops []models.Stop, index int) string {
	if index < 0 || index >= len(stops) {
		return ""
	}
	current := stops[index]
	var sumLat, sumLon float64
	samples := 0
	prevLat, prevLon := current.Latitude, current.Longitude
	for i := index + 1; i < len(stops) && samples < 4; i++ {
		next := stops[i]
		if next.StopId == current.StopId {
			continue
		}
		sumLat += next.Latitude - prevLat
		sumLon += next.Longitude - prevLon
		prevLat, prevLon = next.Latitude, next.Longitude
		samples++
	}
	if samples == 0 {
		return ""
	}
	if abs(sumLat) >= abs(sumLon) {
		if sumLat >= 0 {
			return "N"
		}
		return "S"
	}
	if sumLon >= 0 {
		return "E"
	}
	return "W"
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func summarizeVehicles(vehicles []models.Vehicle, targetStopID int, stopMap map[int]models.Stop) []models.ApproachingVehicle {
	out := make([]models.ApproachingVehicle, 0, len(vehicles))
	for _, vehicle := range vehicles {
		currentStop := ""
		if stop, ok := stopMap[vehicle.StopId]; ok {
			currentStop = stop.Name
		}
		out = append(out, models.ApproachingVehicle{
			VehicleId:                  vehicle.VehicleId,
			Name:                       vehicle.Name,
			Direction:                  vehicle.Direction,
			DirectionLong:              vehicle.DirectionLong,
			CurrentStop:                currentStop,
			LastStop:                   vehicle.LastStop,
			DistanceMiles:              distanceToTargetStop(vehicle, targetStopID, stopMap),
			Deviation:                  vehicle.Deviation,
			DisplayStatus:              vehicle.DisplayStatus,
			OccupancyStatusReportLabel: vehicle.OccupancyStatusReportLabel,
		})
	}
	sortVehiclesForStop(out)
	return out
}

func findDirectionVehicles(orderedStops []models.Stop, vehicles []models.Vehicle, targetStopID int, directionLabel string, stopMap map[int]models.Stop) []models.ApproachingVehicle {
	dirCode := directionCodeFromLabel(directionLabel)
	if dirCode == "" {
		return nil
	}

	out := make([]models.ApproachingVehicle, 0)
	for _, vehicle := range vehicles {
		if vehicleMatchesDirection(vehicle, directionLabel, dirCode) {
			out = append(out, vehicleSummary(vehicle, targetStopID, stopMap))
			continue
		}

		currentIndexes := indexesForStopID(orderedStops, vehicle.StopId)
		for _, idx := range currentIndexes {
			if inferDirectionFromOrderedStops(orderedStops, idx) == dirCode {
				out = append(out, vehicleSummary(vehicle, targetStopID, stopMap))
				break
			}
		}
	}
	dedupeVehicleSummaries(&out)
	sortVehiclesForStop(out)
	return out
}

func vehicleSummary(vehicle models.Vehicle, targetStopID int, stopMap map[int]models.Stop) models.ApproachingVehicle {
	currentStop := ""
	if stop, ok := stopMap[vehicle.StopId]; ok {
		currentStop = stop.Name
	}
	return models.ApproachingVehicle{
		VehicleId:                  vehicle.VehicleId,
		Name:                       vehicle.Name,
		Direction:                  vehicle.Direction,
		DirectionLong:              vehicle.DirectionLong,
		CurrentStop:                currentStop,
		LastStop:                   vehicle.LastStop,
		StopsAway:                  -1,
		DistanceMiles:              distanceToTargetStop(vehicle, targetStopID, stopMap),
		Deviation:                  vehicle.Deviation,
		DisplayStatus:              vehicle.DisplayStatus,
		OccupancyStatusReportLabel: vehicle.OccupancyStatusReportLabel,
	}
}

func dedupeVehicleSummaries(vehicles *[]models.ApproachingVehicle) {
	seen := make(map[int]bool, len(*vehicles))
	out := make([]models.ApproachingVehicle, 0, len(*vehicles))
	for _, vehicle := range *vehicles {
		if seen[vehicle.VehicleId] {
			continue
		}
		seen[vehicle.VehicleId] = true
		out = append(out, vehicle)
	}
	*vehicles = out
}

func sortVehiclesForStop(vehicles []models.ApproachingVehicle) {
	sort.Slice(vehicles, func(i, j int) bool {
		if vehicles[i].StopsAway != vehicles[j].StopsAway {
			if vehicles[i].StopsAway < 0 {
				return false
			}
			if vehicles[j].StopsAway < 0 {
				return true
			}
			if vehicles[i].StopsAway == 0 {
				return true
			}
			if vehicles[j].StopsAway == 0 {
				return false
			}
			return vehicles[i].StopsAway < vehicles[j].StopsAway
		}
		if vehicles[i].DistanceMiles != vehicles[j].DistanceMiles {
			return vehicles[i].DistanceMiles < vehicles[j].DistanceMiles
		}
		if vehicles[i].Deviation != vehicles[j].Deviation {
			return vehicles[i].Deviation < vehicles[j].Deviation
		}
		return vehicles[i].VehicleId < vehicles[j].VehicleId
	})
}

func stopInRouteDetail(detail *models.RouteDetail, stopID int) bool {
	for _, routeStop := range detail.RouteStops {
		if routeStop.StopId == stopID {
			return true
		}
	}
	return false
}

func routeContainsStop(stops []models.Stop, stopID int) bool {
	for _, stop := range stops {
		if stop.StopId == stopID {
			return true
		}
	}
	return false
}

func distanceToTargetStop(vehicle models.Vehicle, targetStopID int, stopMap map[int]models.Stop) float64 {
	stop, ok := stopMap[targetStopID]
	if !ok {
		return 0
	}
	return haversineMiles(vehicle.Latitude, vehicle.Longitude, stop.Latitude, stop.Longitude)
}

func haversineMiles(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMiles = 3958.8
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMiles * c
}

func FormatMessageSummary(message models.Message) string {
	text := strings.TrimSpace(message.Header)
	if text == "" {
		text = strings.TrimSpace(message.Message)
	}
	return text
}

type RouteStops struct {
	Route models.Route  `json:"route"`
	Stops []models.Stop `json:"stops"`
}

type StopDepartures struct {
	Stop           models.Stop                     `json:"stop"`
	Board          models.DepartureBoard           `json:"board"`
	EnrichedGroups []models.EnrichedDepartureGroup `json:"enriched_groups,omitempty"`
}
