package models

import "encoding/json"

type RouteDetail struct {
	Color                    string           `json:"Color"`
	Directions               []RouteDirection `json:"Directions"`
	GoogleDescription        string           `json:"GoogleDescription"`
	Group                    json.RawMessage  `json:"Group"`
	IncludeInGoogle          bool             `json:"IncludeInGoogle"`
	IsHeadway                bool             `json:"IsHeadway"`
	IsHeadwayMonitored       bool             `json:"IsHeadwayMonitored"`
	IsVisible                bool             `json:"IsVisible"`
	IvrDescription           string           `json:"IvrDescription"`
	LongName                 string           `json:"LongName"`
	Messages                 []Message        `json:"Messages"`
	RouteAbbreviation        string           `json:"RouteAbbreviation"`
	RouteId                  int              `json:"RouteId"`
	RouteRecordId            int              `json:"RouteRecordId"`
	RouteStops               []RouteStop      `json:"RouteStops"`
	RouteTraceFilename       string           `json:"RouteTraceFilename"`
	RouteTraceHash64         json.RawMessage  `json:"RouteTraceHash64"`
	ShortName                string           `json:"ShortName"`
	SortOrder                int              `json:"SortOrder"`
	Stops                    json.RawMessage  `json:"Stops"`
	TextColor                string           `json:"TextColor"`
	Vehicles                 []Vehicle        `json:"Vehicles"`
	DetourActiveMessageCount int              `json:"DetourActiveMessageCount"`
}

type RouteDirection struct {
	Dir                   string  `json:"Dir"`
	DirectionDesc         *string `json:"DirectionDesc"`
	DirectionIconFileName *string `json:"DirectionIconFileName"`
}

type RouteStop struct {
	Direction string `json:"Direction"`
	RouteId   int    `json:"RouteId"`
	SortOrder int    `json:"SortOrder"`
	StopId    int    `json:"StopId"`
}
