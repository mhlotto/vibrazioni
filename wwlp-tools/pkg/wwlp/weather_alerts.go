package wwlp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const weatherAlertsBaseURL = "https://weather.psg.nexstardigital.net"
const weatherAlertsEndpoint = "/service/api/v3/alerts/getLiveAlertsByCounties"

type WeatherAlert struct {
	Type               string             `json:"type"`
	WeatherDetail      WeatherAlertDetail `json:"weatherDetail"`
	AlertKey           string             `json:"alertKey"`
	EffectiveTimestamp string             `json:"effectiveTimestamp"`
	ExpireTimestamp    string             `json:"expireTimestamp"`
	CreateTimestamp    string             `json:"createTimestamp"`
	Description        string             `json:"description"`
	Severity           string             `json:"severity"`
	Phenomena          string             `json:"phenomena"`
	AreaID             string             `json:"areaId"`
	AreaName           string             `json:"areaName"`
}

type WeatherAlertDetail struct {
	Payload            string `json:"payload"`
	LongDescription    string `json:"longDescription"`
	AlertType          string `json:"alertType"`
	AreaName           string `json:"areaName"`
	EffectiveTimestamp string `json:"effectiveTimestamp"`
	ExpireTimestamp    string `json:"expireTimestamp"`
}

type WeatherAlertPayload struct {
	AreaName         string             `json:"areaName"`
	EventDescription string             `json:"eventDescription"`
	HeadlineText     string             `json:"headlineText"`
	HeadlineTextAlt  string             `json:"headline_text"`
	EffectiveTime    string             `json:"effectiveTimeLocal"`
	ExpireTime       string             `json:"expireTimeLocal"`
	EndTime          string             `json:"endTimeLocal"`
	Severity         string             `json:"severity"`
	Urgency          string             `json:"urgency"`
	Certainty        string             `json:"certainty"`
	Texts            []WeatherAlertText `json:"texts"`
}

type WeatherAlertText struct {
	Description  string `json:"description"`
	LanguageCode string `json:"languageCode"`
}

func LoadWeatherAlerts(r io.Reader) ([]WeatherAlert, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read alerts: %w", err)
	}
	var alerts []WeatherAlert
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, fmt.Errorf("decode alerts: %w", err)
	}
	return alerts, nil
}

func LoadWeatherAlertsFile(path string) ([]WeatherAlert, error) {
	if path == "-" {
		return LoadWeatherAlerts(os.Stdin)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	return LoadWeatherAlerts(f)
}

func LoadWeatherAlertsURL(counties string) ([]WeatherAlert, error) {
	u, err := url.Parse(weatherAlertsBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	u.Path = weatherAlertsEndpoint
	if counties != "" {
		q := u.Query()
		q.Set("counties", counties)
		u.RawQuery = q.Encode()
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", "https://www.wwlp.com")
	req.Header.Set("Referer", "https://www.wwlp.com/")
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status: %s", resp.Status)
	}
	return LoadWeatherAlerts(resp.Body)
}

func ParseWeatherAlertPayload(payload string) (*WeatherAlertPayload, error) {
	if payload == "" {
		return nil, nil
	}
	var out WeatherAlertPayload
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return &out, nil
}
