package models

type DepartureBoard struct {
	StopName    string           `json:"stop_name"`
	LastUpdated string           `json:"last_updated,omitempty"`
	Groups      []DepartureGroup `json:"groups"`
}

type DepartureGroup struct {
	RouteAndDirection string   `json:"route_and_direction"`
	Times             []string `json:"times"`
}

type ApproachingVehicle struct {
	VehicleId                  int     `json:"vehicle_id"`
	Name                       string  `json:"name"`
	Direction                  string  `json:"direction"`
	DirectionLong              string  `json:"direction_long"`
	CurrentStop                string  `json:"current_stop,omitempty"`
	LastStop                   string  `json:"last_stop,omitempty"`
	StopsAway                  int     `json:"stops_away,omitempty"`
	DistanceMiles              float64 `json:"distance_miles,omitempty"`
	Deviation                  int     `json:"deviation"`
	DisplayStatus              string  `json:"display_status"`
	OccupancyStatusReportLabel string  `json:"occupancy_status_report_label,omitempty"`
}

type EnrichedDepartureGroup struct {
	RouteAndDirection     string               `json:"route_and_direction"`
	Times                 []string             `json:"times"`
	MatchedRouteId        int                  `json:"matched_route_id,omitempty"`
	MatchedRouteShortName string               `json:"matched_route_short_name,omitempty"`
	MatchedRouteLongName  string               `json:"matched_route_long_name,omitempty"`
	StopInRouteSequence   bool                 `json:"stop_in_route_sequence,omitempty"`
	LiveVehicles          []ApproachingVehicle `json:"live_vehicles,omitempty"`
	DirectionVehicles     []ApproachingVehicle `json:"direction_vehicles,omitempty"`
	RouteVehicles         []ApproachingVehicle `json:"route_vehicles,omitempty"`
}
