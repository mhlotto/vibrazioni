package wwlp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func LoadTemplateVars(r io.Reader) (*TemplateVars, error) {
	dec := json.NewDecoder(r)
	var tv TemplateVars
	if err := dec.Decode(&tv); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	return &tv, nil
}

func LoadTemplateVarsFile(path string) (*TemplateVars, error) {
	if path == "-" {
		return LoadTemplateVars(os.Stdin)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	return LoadTemplateVars(f)
}

func LoadTemplateVarsURL(url string) (*TemplateVars, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status: %s", resp.Status)
	}
	return LoadTemplateVars(resp.Body)
}
