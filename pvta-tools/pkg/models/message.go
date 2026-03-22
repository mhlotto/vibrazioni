package models

import "encoding/json"

type Message struct {
	Cause               int               `json:"Cause"`
	CauseReportLabel    string            `json:"CauseReportLabel"`
	Header              string            `json:"Header"`
	ChannelMessages     []json.RawMessage `json:"ChannelMessages"`
	DaysOfWeek          *int              `json:"DaysOfWeek"`
	Effect              int               `json:"Effect"`
	EffectReportLabel   string            `json:"EffectReportLabel"`
	FromDate            string            `json:"FromDate"`
	FromTime            string            `json:"FromTime"`
	Message             string            `json:"Message"`
	MessageId           int               `json:"MessageId"`
	MessageTranslations []json.RawMessage `json:"MessageTranslations"`
	Priority            int               `json:"Priority"`
	PublicAccess        int               `json:"PublicAccess"`
	Published           bool              `json:"Published"`
	Routes              []int             `json:"Routes"`
	Signs               []int             `json:"Signs"`
	ToDate              string            `json:"ToDate"`
	ToTime              string            `json:"ToTime"`
	URL                 *string           `json:"URL"`
	DetourID            json.RawMessage   `json:"Detour_Id"`
	SharedMessageKey    *string           `json:"SharedMessageKey"`
	IsPrimaryRecord     bool              `json:"IsPrimaryRecord"`
}
