package wwlp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func LoadTemplateVars(r io.Reader) (*TemplateVars, []string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("read json: %w", err)
	}
	return LoadTemplateVarsBytes(data)
}

func LoadTemplateVarsFile(path string) (*TemplateVars, []string, error) {
	if path == "-" {
		return LoadTemplateVars(os.Stdin)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	return LoadTemplateVars(f)
}

func LoadTemplateVarsURL(url string) (*TemplateVars, []string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("http status: %s", resp.Status)
	}
	return LoadTemplateVars(resp.Body)
}

func LoadTemplateVarsBytes(data []byte) (*TemplateVars, []string, error) {
	warnings := ValidateTemplateVarsShape(data)

	var tv TemplateVars
	if err := json.Unmarshal(data, &tv); err != nil {
		return nil, warnings, fmt.Errorf("decode json: %w", err)
	}
	return &tv, warnings, nil
}

func ValidateTemplateVarsShape(data []byte) []string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return []string{fmt.Sprintf("unable to inspect JSON shape: %v", err)}
	}

	required := []struct {
		key      string
		jsonType string
	}{
		{key: "top_stories", jsonType: "object"},
		{key: "additional_top_stories", jsonType: "object"},
		{key: "headline_lists", jsonType: "array"},
		{key: "weather", jsonType: "object"},
		{key: "alert_banners", jsonType: "object"},
	}

	var warnings []string
	for _, r := range required {
		msg := raw[r.key]
		if len(msg) == 0 {
			warnings = append(warnings, fmt.Sprintf("missing key: %s", r.key))
			continue
		}
		t := jsonType(msg)
		if t == "null" {
			warnings = append(warnings, fmt.Sprintf("key is null: %s", r.key))
			continue
		}
		if t != r.jsonType {
			warnings = append(warnings, fmt.Sprintf("unexpected type for %s: %s", r.key, t))
		}
	}
	return warnings
}

func jsonType(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return "empty"
	}
	switch s[0] {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "bool"
	case 'n':
		return "null"
	default:
		if (s[0] >= '0' && s[0] <= '9') || s[0] == '-' {
			return "number"
		}
	}
	return "unknown"
}
