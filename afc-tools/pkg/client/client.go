package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/afc-tools/pkg/cache"
	"github.com/mhlotto/vibrazioni/afc-tools/pkg/models"
	"github.com/mhlotto/vibrazioni/afc-tools/pkg/parser"
)

const (
	defaultBaseURL             = "https://www.arsenal.com"
	resultsAndFixturesListPath = "/results-and-fixtures-list?"
	resultsAndFixturesCacheKey = "results-and-fixtures"
	defaultUpgradeInsecure     = "1"
	defaultUserAgent           = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"
	defaultSecCHUA             = "\"Chromium\";v=\"146\", \"Not-A.Brand\";v=\"24\", \"Brave\";v=\"146\""
	defaultSecCHUAMobile       = "?0"
	defaultSecCHUAPlatform     = "\"macOS\""
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

var timeNow = time.Now

func New(baseURL string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) GetResultsAndFixturesList(ctx context.Context, dataCache *cache.Cache) (*models.ResultsAndFixturesList, error) {
	if dataCache != nil {
		cached, err := dataCache.Get(resultsAndFixturesCacheKey)
		if err == nil {
			var parsed models.ResultsAndFixturesList
			if err := json.Unmarshal(cached, &parsed); err == nil {
				return &parsed, nil
			}
		} else if !errors.Is(err, cache.ErrNotFound) && !errors.Is(err, cache.ErrStale) {
			return nil, fmt.Errorf("read results-and-fixtures cache: %w", err)
		}
	}

	body, err := c.getResultsAndFixturesListHTML(ctx)
	if err != nil {
		return nil, err
	}

	parsed, err := parser.ParseResultsAndFixturesList(body)
	if err != nil {
		return nil, fmt.Errorf("parse results and fixtures list: %w", err)
	}

	if dataCache != nil {
		encoded, err := json.Marshal(parsed)
		if err != nil {
			return nil, fmt.Errorf("marshal results-and-fixtures cache payload: %w", err)
		}
		if err := dataCache.Put(resultsAndFixturesCacheKey, encoded); err != nil {
			return nil, fmt.Errorf("write results-and-fixtures cache: %w", err)
		}
	}

	return parsed, nil
}

func (c *Client) getResultsAndFixturesListHTML(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+resultsAndFixturesListPath, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Upgrade-Insecure-Requests", defaultUpgradeInsecure)
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("sec-ch-ua", defaultSecCHUA)
	req.Header.Set("sec-ch-ua-mobile", defaultSecCHUAMobile)
	req.Header.Set("sec-ch-ua-platform", defaultSecCHUAPlatform)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return body, nil
}

// GetResultsAndFixtureslist preserves the requested exported routine name.
func (c *Client) GetResultsAndFixtureslist(ctx context.Context, dataCache *cache.Cache) (*models.ResultsAndFixturesList, error) {
	return c.GetResultsAndFixturesList(ctx, dataCache)
}

func (c *Client) GetFixturesUpToDate(ctx context.Context, upTo time.Time, dataCache *cache.Cache) ([]models.Match, error) {
	results, err := c.GetResultsAndFixturesList(ctx, dataCache)
	if err != nil {
		return nil, err
	}

	now := timeNow()
	if upTo.Before(now) {
		return []models.Match{}, nil
	}

	fixtures := make([]models.Match, 0)
	for _, match := range results.Matches {
		if match.Status != models.MatchStatusScheduled {
			continue
		}
		if match.Kickoff.Before(now) {
			continue
		}
		if match.Kickoff.After(upTo) {
			continue
		}
		fixtures = append(fixtures, match)
	}

	return fixtures, nil
}

func (c *Client) GetFixturesWithinNDays(ctx context.Context, days int, dataCache *cache.Cache) ([]models.Match, error) {
	if days <= 0 {
		return []models.Match{}, nil
	}

	upTo := timeNow().Add(time.Duration(days) * 24 * time.Hour)
	return c.GetFixturesUpToDate(ctx, upTo, dataCache)
}

func (c *Client) GetPastFixturesWithinNDays(ctx context.Context, days int, dataCache *cache.Cache) ([]models.Match, error) {
	if days <= 0 {
		return []models.Match{}, nil
	}

	results, err := c.GetResultsAndFixturesList(ctx, dataCache)
	if err != nil {
		return nil, err
	}

	now := timeNow()
	from := now.Add(-time.Duration(days) * 24 * time.Hour)
	fixtures := make([]models.Match, 0)
	for _, match := range results.Matches {
		if match.Status != models.MatchStatusFinished {
			continue
		}
		if match.Kickoff.Before(from) {
			continue
		}
		if match.Kickoff.After(now) {
			continue
		}
		fixtures = append(fixtures, match)
	}

	return fixtures, nil
}
