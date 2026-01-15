package wwlp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var jsonLDScriptRe = regexp.MustCompile(`(?is)<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
var forecastHeadingRe = regexp.MustCompile(`(?i)\s+((MONDAY|TUESDAY|WEDNESDAY|THURSDAY|FRIDAY|SATURDAY|SUNDAY)(?: NIGHT)?\s*:)`)
var forecastTodayRe = regexp.MustCompile(`(?i)\s+((TODAY|TONIGHT)\s*:)`)
var forecastNumberWordRe = regexp.MustCompile(`(\d)(Highs|Lows|Winds)`)
var forecastWordRe = regexp.MustCompile(`([A-Za-z])((Highs|Lows|Winds):)`)

func LoadForecastDiscussion(r io.Reader) (*ForecastDiscussion, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	return LoadForecastDiscussionBytes(data)
}

func LoadForecastDiscussionFile(path string) (*ForecastDiscussion, error) {
	if path == "-" {
		return LoadForecastDiscussion(os.Stdin)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	return LoadForecastDiscussion(f)
}

func LoadForecastDiscussionBytes(data []byte) (*ForecastDiscussion, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty input")
	}
	if trimmed[0] == '{' {
		article, ok, err := parseForecastDiscussionJSON(trimmed)
		if err != nil {
			return nil, fmt.Errorf("decode json: %w", err)
		}
		if !ok {
			return nil, fmt.Errorf("can't find forecast json-ld")
		}
		return article, nil
	}
	for _, block := range extractJSONLDBlocks(trimmed) {
		article, ok, err := parseForecastDiscussionJSON([]byte(block))
		if err != nil {
			continue
		}
		if ok {
			return article, nil
		}
	}
	return nil, fmt.Errorf("can't find forecast json-ld")
}

func LoadForecastDiscussionURL(url string) (*ForecastDiscussion, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status: %s", resp.Status)
	}
	return LoadForecastDiscussion(resp.Body)
}

func extractJSONLDBlocks(data []byte) []string {
	matches := jsonLDScriptRe.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		block := strings.TrimSpace(string(m[1]))
		if block != "" {
			out = append(out, block)
		}
	}
	return out
}

func parseForecastDiscussionJSON(data []byte) (*ForecastDiscussion, bool, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false, err
	}
	if !matchesForecastDiscussion(raw) {
		return nil, false, nil
	}
	return forecastDiscussionFromMap(raw), true, nil
}

func CleanForecastText(text string) string {
	text = html.UnescapeString(text)
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = strings.Join(strings.Fields(text), " ")
	text = forecastNumberWordRe.ReplaceAllString(text, "$1 $2")
	text = forecastWordRe.ReplaceAllString(text, "$1 $2")
	text = forecastHeadingRe.ReplaceAllString(text, "\n$1")
	text = forecastTodayRe.ReplaceAllString(text, "\n$1")
	return strings.TrimSpace(text)
}

func matchesForecastDiscussion(raw map[string]any) bool {
	if t := stringFromAny(raw["@type"]); t != "" && !strings.EqualFold(t, "NewsArticle") {
		return false
	}
	if v, ok := raw["articleSelection"]; ok {
		return strings.EqualFold(strings.TrimSpace(stringFromAny(v)), "Today's Forecast")
	}
	for _, g := range stringSliceFromAny(raw["genre"]) {
		if strings.EqualFold(g, "Weather News") || strings.EqualFold(g, "Today's Forecast") {
			return true
		}
	}
	return false
}

func forecastDiscussionFromMap(raw map[string]any) *ForecastDiscussion {
	return &ForecastDiscussion{
		Headline:       stringFromAny(raw["headline"]),
		Description:    stringFromAny(raw["description"]),
		URL:            stringFromAny(raw["url"]),
		DatePublished:  stringFromAny(raw["datePublished"]),
		DateModified:   stringFromAny(raw["dateModified"]),
		ArticleBody:    stringFromAny(raw["articleBody"]),
		ArticleSection: stringFromAny(raw["articleSection"]),
		Genre:          stringSliceFromAny(raw["genre"]),
		Authors:        authorNamesFromAny(raw["author"]),
	}
}

func stringFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func stringSliceFromAny(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{t}
	default:
		return nil
	}
}

func authorNamesFromAny(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if name := authorNameFromAny(item); name != "" {
				out = append(out, name)
			}
		}
		return out
	case map[string]any:
		if name := stringFromAny(t["name"]); name != "" {
			return []string{name}
		}
	case string:
		return []string{t}
	}
	return nil
}

func authorNameFromAny(v any) string {
	switch t := v.(type) {
	case map[string]any:
		return stringFromAny(t["name"])
	case string:
		return t
	default:
		return ""
	}
}
