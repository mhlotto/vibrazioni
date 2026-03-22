package models

import "encoding/json"

type Vehicle struct {
	BlockFareboxId             int             `json:"BlockFareboxId"`
	CommStatus                 string          `json:"CommStatus"`
	Destination                string          `json:"Destination"`
	Deviation                  int             `json:"Deviation"`
	Direction                  string          `json:"Direction"`
	DirectionLong              string          `json:"DirectionLong"`
	DisplayStatus              string          `json:"DisplayStatus"`
	StopId                     int             `json:"StopId"`
	CurrentStatus              json.RawMessage `json:"CurrentStatus"`
	DriverName                 json.RawMessage `json:"DriverName"`
	DriverLastName             json.RawMessage `json:"DriverLastName"`
	DriverFirstName            json.RawMessage `json:"DriverFirstName"`
	DriverFareboxId            int             `json:"DriverFareboxId"`
	VehicleFareboxId           int             `json:"VehicleFareboxId"`
	GPSStatus                  int             `json:"GPSStatus"`
	Heading                    int             `json:"Heading"`
	LastStop                   string          `json:"LastStop"`
	LastUpdated                string          `json:"LastUpdated"`
	Latitude                   float64         `json:"Latitude"`
	Longitude                  float64         `json:"Longitude"`
	Name                       string          `json:"Name"`
	OccupancyStatus            int             `json:"OccupancyStatus"`
	OnBoard                    int             `json:"OnBoard"`
	OpStatus                   string          `json:"OpStatus"`
	RouteId                    int             `json:"RouteId"`
	RunId                      int             `json:"RunId"`
	Speed                      json.RawMessage `json:"Speed"`
	TripId                     int             `json:"TripId"`
	VehicleId                  int             `json:"VehicleId"`
	SeatingCapacity            int             `json:"SeatingCapacity"`
	TotalCapacity              int             `json:"TotalCapacity"`
	PropertyName               string          `json:"PropertyName"`
	OccupancyStatusReportLabel string          `json:"OccupancyStatusReportLabel"`
}
