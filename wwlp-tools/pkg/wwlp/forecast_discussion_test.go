package wwlp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadForecastDiscussionFromHTML(t *testing.T) {
	path := filepath.Join("..", "..", "natlang-forecast.out")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}
	article, err := LoadForecastDiscussionBytes(data)
	if err != nil {
		t.Fatalf("load discussion: %v", err)
	}
	if !strings.Contains(article.ArticleBody, "Good Thursday morning.") {
		t.Fatalf("expected article body content, got: %q", article.ArticleBody)
	}
}

func TestLoadForecastDiscussionFromJSON(t *testing.T) {
	path := filepath.Join("..", "..", "natlang-forecast-desiredblob.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}
	article, err := LoadForecastDiscussionBytes(data)
	if err != nil {
		t.Fatalf("load discussion: %v", err)
	}
	if !strings.Contains(article.ArticleBody, "Good Thursday morning.") {
		t.Fatalf("expected article body content, got: %q", article.ArticleBody)
	}
}

func TestLoadForecastDiscussionMissing(t *testing.T) {
	data := []byte(`<script type="application/ld+json">{"@type":"NewsArticle","articleSection":"Other","articleBody":"Nope"}</script>`)
	_, err := LoadForecastDiscussionBytes(data)
	if err == nil || !strings.Contains(err.Error(), "can't find") {
		t.Fatalf("expected can't find error, got: %v", err)
	}
}

func TestLoadForecastDiscussionGenreFallback(t *testing.T) {
	data := []byte(`{"@type":"NewsArticle","genre":["Weather News"],"articleBody":"Hello"}`)
	article, err := LoadForecastDiscussionBytes(data)
	if err != nil {
		t.Fatalf("load discussion: %v", err)
	}
	if article.ArticleBody != "Hello" {
		t.Fatalf("expected article body, got: %q", article.ArticleBody)
	}
}

func TestCleanForecastText(t *testing.T) {
	input := "THURSDAY: AM Clouds &amp; Showers Highs: 42-46Winds: S to W 10-15 MPH"
	got := CleanForecastText(input)
	if !strings.Contains(got, "Clouds & Showers") {
		t.Fatalf("expected HTML entities to be unescaped, got: %q", got)
	}
	if !strings.Contains(got, "Highs: 42-46 Winds:") {
		t.Fatalf("expected spacing cleanup, got: %q", got)
	}
}
