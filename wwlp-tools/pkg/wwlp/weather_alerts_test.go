package wwlp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWeatherAlertsAndPayload(t *testing.T) {
	path := filepath.Join("..", "..", "weather-alerts-response.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}
	alerts, err := LoadWeatherAlerts(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("load alerts: %v", err)
	}
	if len(alerts) == 0 {
		t.Fatalf("expected alerts, got none")
	}
	payload, err := ParseWeatherAlertPayload(alerts[0].WeatherDetail.Payload)
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	if payload == nil || payload.AreaName == "" {
		t.Fatalf("expected payload area name")
	}
}
