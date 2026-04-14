package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/afc-tools/pkg/cache"
)

func TestGetResultsAndFixturesList(t *testing.T) {
	var gotPath string
	var gotHeaders http.Header
	fixturePath := filepath.Join("testdata", "results-and-fixtures-get.out")
	fixtureBody, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", fixturePath, err)
	}

	client := New("https://www.arsenal.com")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotPath = r.URL.RequestURI()
			gotHeaders = r.Header.Clone()
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(string(fixtureBody))),
			}, nil
		}),
	}
	got, err := client.GetResultsAndFixturesList(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetResultsAndFixturesList: %v", err)
	}

	if got.Title != "Results And Fixtures List | Arsenal.com" {
		t.Fatalf("unexpected title: %q", got.Title)
	}
	if len(got.Matches) == 0 {
		t.Fatal("expected parsed matches")
	}
	if gotPath != "/results-and-fixtures-list?" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
	if gotHeaders.Get("Upgrade-Insecure-Requests") != "1" {
		t.Fatalf("unexpected Upgrade-Insecure-Requests: %q", gotHeaders.Get("Upgrade-Insecure-Requests"))
	}
	if gotHeaders.Get("User-Agent") != defaultUserAgent {
		t.Fatalf("unexpected User-Agent: %q", gotHeaders.Get("User-Agent"))
	}
	if gotHeaders.Get("sec-ch-ua") != defaultSecCHUA {
		t.Fatalf("unexpected sec-ch-ua: %q", gotHeaders.Get("sec-ch-ua"))
	}
	if gotHeaders.Get("sec-ch-ua-mobile") != defaultSecCHUAMobile {
		t.Fatalf("unexpected sec-ch-ua-mobile: %q", gotHeaders.Get("sec-ch-ua-mobile"))
	}
	if gotHeaders.Get("sec-ch-ua-platform") != defaultSecCHUAPlatform {
		t.Fatalf("unexpected sec-ch-ua-platform: %q", gotHeaders.Get("sec-ch-ua-platform"))
	}
}

func TestGetResultsAndFixtureslistAlias(t *testing.T) {
	fixturePath := filepath.Join("testdata", "results-and-fixtures-get.out")
	fixtureBody, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", fixturePath, err)
	}

	client := New("https://www.arsenal.com")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(string(fixtureBody))),
			}, nil
		}),
	}
	got, err := client.GetResultsAndFixtureslist(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetResultsAndFixtureslist: %v", err)
	}
	if got.Title != "Results And Fixtures List | Arsenal.com" {
		t.Fatalf("unexpected title: %q", got.Title)
	}
}

func TestGetResultsAndFixturesListUsesFreshCache(t *testing.T) {
	fixturePath := filepath.Join("testdata", "results-and-fixtures.json")
	fixtureBody, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", fixturePath, err)
	}

	dataCache := cache.New(t.TempDir())
	if err := dataCache.Put(resultsAndFixturesCacheKey, fixtureBody); err != nil {
		t.Fatalf("cache Put(): %v", err)
	}

	client := New("https://www.arsenal.com")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			t.Fatal("unexpected network call with fresh cache")
			return nil, nil
		}),
	}

	got, err := client.GetResultsAndFixturesList(context.Background(), dataCache)
	if err != nil {
		t.Fatalf("GetResultsAndFixturesList: %v", err)
	}

	if got.Title != "Results And Fixtures List | Arsenal.com" {
		t.Fatalf("unexpected title: %q", got.Title)
	}
}

func TestGetResultsAndFixturesListStoresParsedJSONInCache(t *testing.T) {
	fixturePath := filepath.Join("testdata", "results-and-fixtures-get.out")
	fixtureBody, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", fixturePath, err)
	}

	dataCache := cache.New(t.TempDir())
	client := New("https://www.arsenal.com")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(string(fixtureBody))),
			}, nil
		}),
	}

	got, err := client.GetResultsAndFixturesList(context.Background(), dataCache)
	if err != nil {
		t.Fatalf("GetResultsAndFixturesList: %v", err)
	}

	cached, err := dataCache.Get(resultsAndFixturesCacheKey)
	if err != nil {
		t.Fatalf("cache Get(): %v", err)
	}

	var cachedParsed struct {
		Title   string `json:"title"`
		Matches []any  `json:"matches"`
	}
	if err := json.Unmarshal(cached, &cachedParsed); err != nil {
		t.Fatalf("json.Unmarshal(cache): %v", err)
	}

	if cachedParsed.Title != got.Title {
		t.Fatalf("cached title = %q, want %q", cachedParsed.Title, got.Title)
	}
	if len(cachedParsed.Matches) != len(got.Matches) {
		t.Fatalf("cached matches len = %d, want %d", len(cachedParsed.Matches), len(got.Matches))
	}
}

func TestGetFixturesUpToDate(t *testing.T) {
	fixturePath := filepath.Join("testdata", "results-and-fixtures-get.out")
	fixtureBody, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", fixturePath, err)
	}

	client := New("https://www.arsenal.com")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(string(fixtureBody))),
			}, nil
		}),
	}

	fixedNow := time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
	oldNow := timeNow
	timeNow = func() time.Time { return fixedNow }
	defer func() { timeNow = oldNow }()

	upTo := time.Date(2026, time.April, 30, 23, 59, 59, 0, time.UTC)
	got, err := client.GetFixturesUpToDate(context.Background(), upTo, nil)
	if err != nil {
		t.Fatalf("GetFixturesUpToDate: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("len(GetFixturesUpToDate()) = %d, want 3", len(got))
	}

	if got[0].HomeTeam != "Arsenal" || got[0].AwayTeam != "Sporting CP" {
		t.Fatalf("got first fixture = %s vs %s, want Arsenal vs Sporting CP", got[0].HomeTeam, got[0].AwayTeam)
	}

	if got[1].HomeTeam != "Manchester City" || got[1].AwayTeam != "Arsenal" {
		t.Fatalf("got second fixture = %s vs %s, want Manchester City vs Arsenal", got[1].HomeTeam, got[1].AwayTeam)
	}

	if got[2].HomeTeam != "Arsenal" || got[2].AwayTeam != "Newcastle United" {
		t.Fatalf("got third fixture = %s vs %s, want Arsenal vs Newcastle United", got[2].HomeTeam, got[2].AwayTeam)
	}

	for _, match := range got {
		if match.Status != "scheduled" {
			t.Fatalf("fixture status = %q, want scheduled", match.Status)
		}
		if match.Kickoff.Before(fixedNow) || match.Kickoff.After(upTo) {
			t.Fatalf("fixture kickoff %s outside range [%s, %s]", match.Kickoff, fixedNow, upTo)
		}
	}
}

func TestGetFixturesUpToDateEmptyWhenUpperBoundBeforeNow(t *testing.T) {
	client := New("https://www.arsenal.com")
	fixedNow := time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
	oldNow := timeNow
	timeNow = func() time.Time { return fixedNow }
	defer func() { timeNow = oldNow }()

	dataCache := cache.New(t.TempDir())
	if err := dataCache.Put(resultsAndFixturesCacheKey, []byte(`{"title":"x","matches":[]}`)); err != nil {
		t.Fatalf("cache Put(): %v", err)
	}

	got, err := client.GetFixturesUpToDate(context.Background(), fixedNow.Add(-time.Minute), dataCache)
	if err != nil {
		t.Fatalf("GetFixturesUpToDate: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("len(GetFixturesUpToDate()) = %d, want 0", len(got))
	}
}

func TestGetFixturesWithinNDays(t *testing.T) {
	fixturePath := filepath.Join("testdata", "results-and-fixtures-get.out")
	fixtureBody, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", fixturePath, err)
	}

	client := New("https://www.arsenal.com")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(string(fixtureBody))),
			}, nil
		}),
	}

	fixedNow := time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
	oldNow := timeNow
	timeNow = func() time.Time { return fixedNow }
	defer func() { timeNow = oldNow }()

	got, err := client.GetFixturesWithinNDays(context.Background(), 14, nil)
	if err != nil {
		t.Fatalf("GetFixturesWithinNDays: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("len(GetFixturesWithinNDays()) = %d, want 3", len(got))
	}

	if got[0].HomeTeam != "Arsenal" || got[0].AwayTeam != "Sporting CP" {
		t.Fatalf("got first fixture = %s vs %s, want Arsenal vs Sporting CP", got[0].HomeTeam, got[0].AwayTeam)
	}
}

func TestGetFixturesWithinNDaysNonPositive(t *testing.T) {
	client := New("https://www.arsenal.com")
	fixedNow := time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
	oldNow := timeNow
	timeNow = func() time.Time { return fixedNow }
	defer func() { timeNow = oldNow }()

	got, err := client.GetFixturesWithinNDays(context.Background(), 0, nil)
	if err != nil {
		t.Fatalf("GetFixturesWithinNDays: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("len(GetFixturesWithinNDays()) = %d, want 0", len(got))
	}
}

func TestGetPastFixturesWithinNDays(t *testing.T) {
	fixturePath := filepath.Join("testdata", "results-and-fixtures-get.out")
	fixtureBody, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", fixturePath, err)
	}

	client := New("https://www.arsenal.com")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(string(fixtureBody))),
			}, nil
		}),
	}

	fixedNow := time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
	oldNow := timeNow
	timeNow = func() time.Time { return fixedNow }
	defer func() { timeNow = oldNow }()

	got, err := client.GetPastFixturesWithinNDays(context.Background(), 7, nil)
	if err != nil {
		t.Fatalf("GetPastFixturesWithinNDays: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(GetPastFixturesWithinNDays()) = %d, want 2", len(got))
	}

	if got[0].HomeTeam != "Sporting CP" || got[0].AwayTeam != "Arsenal" {
		t.Fatalf("got first past fixture = %s vs %s, want Sporting CP vs Arsenal", got[0].HomeTeam, got[0].AwayTeam)
	}

	if got[1].HomeTeam != "Arsenal" || got[1].AwayTeam != "Bournemouth" {
		t.Fatalf("got second past fixture = %s vs %s, want Arsenal vs Bournemouth", got[1].HomeTeam, got[1].AwayTeam)
	}
}

func TestGetPastFixturesWithinNDaysNonPositive(t *testing.T) {
	client := New("https://www.arsenal.com")

	got, err := client.GetPastFixturesWithinNDays(context.Background(), 0, nil)
	if err != nil {
		t.Fatalf("GetPastFixturesWithinNDays: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("len(GetPastFixturesWithinNDays()) = %d, want 0", len(got))
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
