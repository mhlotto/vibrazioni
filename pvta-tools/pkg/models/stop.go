package models

type Stop struct {
	Description  string  `json:"Description"`
	IsTimePoint  bool    `json:"IsTimePoint"`
	Latitude     float64 `json:"Latitude"`
	Longitude    float64 `json:"Longitude"`
	Name         string  `json:"Name"`
	StopId       int     `json:"StopId"`
	StopRecordId int     `json:"StopRecordId"`
}
