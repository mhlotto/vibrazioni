package wwlp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClothingProfileByKey(t *testing.T) {
	profile, err := ClothingProfileByKey("mid40-male")
	if err != nil {
		t.Fatalf("expected default profile, got error: %v", err)
	}
	if profile.DisplayName != "Mid-40s Male" {
		t.Fatalf("unexpected profile: %+v", profile)
	}

	if _, err := ClothingProfileByKey("unknown"); err == nil {
		t.Fatalf("expected unknown profile error")
	}

	profile, err = ClothingProfileByKey("mid40-female")
	if err != nil {
		t.Fatalf("expected mid40-female profile, got error: %v", err)
	}
	if profile.DisplayName != "Mid-40s Female" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestBuildClothingPrompt(t *testing.T) {
	weather := &Weather{
		ThreeDay: &ThreeDayWeather{
			Current:  &WeatherPoint{Temperature: "48", Phrase: "Cloudy", PrecipChance: "10"},
			Tonight:  &WeatherPoint{Temperature: "39", Phrase: "Light rain", PrecipChance: "40"},
			Tomorrow: &WeatherPoint{Temperature: "55", Phrase: "Breezy", PrecipChance: "0"},
		},
		Hourly: []HourlyForecast{
			{Time: "9 AM", Temperature: "47", LongPhrase: "Cloudy"},
			{Time: "12 PM", Temperature: "50", LongPhrase: "Breezy"},
		},
		SevenDay: []DailyForecast{
			{
				DayOfWeek:      "Sunday",
				MaxTemperature: "55",
				MinTemperature: "38",
				ShortPhrase:    "Breezy",
				PrecipChance:   "20",
				DayNarrative:   "Cool and windy through the afternoon",
			},
		},
	}

	profile, err := ClothingProfileByKey(DefaultClothingProfileKey)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	profile.Activity = "walking to work and standing outside for 20 minutes"
	profile.ExtraNotes = "runs cold and prefers a hat"
	prompt := buildClothingPrompt(weather, profile, ClothingOptions{})

	for _, want := range []string{
		"Mid-40s Male",
		"Expected activity: walking to work and standing outside for 20 minutes.",
		"Extra wearer notes: runs cold and prefers a hat.",
		"Current: 48F, Cloudy with 10% precip chance.",
		"Tonight: 39F, Light rain with 40% precip chance.",
		"Daily outlook: Sunday high 55F low 38F, Breezy with 20% precip chance.",
		"Near-term hourly: 9 AM 47F Cloudy; 12 PM 50F Breezy.",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\nprompt=%s", want, prompt)
		}
	}
}

func TestRecommendClothes(t *testing.T) {
	var captured chatCompletionsRequest
	clientHTTP := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/chat/completions" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(bytes.NewBufferString(
					`{"choices":[{"message":{"role":"assistant","content":"Wear jeans, a long-sleeve shirt, and a light jacket."}}]}`,
				)),
			}, nil
		}),
	}

	profile, err := ClothingProfileByKey(DefaultClothingProfileKey)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	profile.Activity = "commuting and light walking"
	profile.ExtraNotes = "gets cold easily"
	client := &LlamaClient{
		BaseURL:    "http://llama.local/v1/chat/completions",
		Model:      "Qwen/Qwen3-4B-GGUF",
		HTTPClient: clientHTTP,
	}
	weather := &Weather{
		ThreeDay: &ThreeDayWeather{
			Current: &WeatherPoint{Temperature: "51", Phrase: "Mostly cloudy"},
		},
	}

	got, err := client.RecommendClothes(weather, profile)
	if err != nil {
		t.Fatalf("recommend clothes: %v", err)
	}
	if !strings.Contains(got, "light jacket") {
		t.Fatalf("unexpected response: %q", got)
	}
	if captured.Model != "Qwen/Qwen3-4B-GGUF" {
		t.Fatalf("unexpected model: %q", captured.Model)
	}
	if len(captured.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(captured.Messages))
	}
	if !strings.Contains(captured.Messages[0].Content, "mid-40s man") {
		t.Fatalf("missing profile guidance in system prompt: %q", captured.Messages[0].Content)
	}
	if !strings.Contains(captured.Messages[0].Content, "Return exactly these lines in order: Today:, Outerwear:, Footwear:, Extras:, Why:.") {
		t.Fatalf("missing structured output guidance in system prompt: %q", captured.Messages[0].Content)
	}
	if !strings.Contains(captured.Messages[0].Content, "Make Why the most detailed line") {
		t.Fatalf("missing Why detail guidance in system prompt: %q", captured.Messages[0].Content)
	}
	if !strings.Contains(captured.Messages[0].Content, "gets cold easily") {
		t.Fatalf("missing extra notes in system prompt: %q", captured.Messages[0].Content)
	}
	if !strings.Contains(captured.Messages[1].Content, "Current: 51F, Mostly cloudy.") {
		t.Fatalf("missing weather summary in user prompt: %q", captured.Messages[1].Content)
	}
	if !strings.Contains(captured.Messages[1].Content, "Expected activity: commuting and light walking.") {
		t.Fatalf("missing activity in user prompt: %q", captured.Messages[1].Content)
	}
}

func TestRecommendClothesWithPoem(t *testing.T) {
	var captured chatCompletionsRequest
	clientHTTP := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(bytes.NewBufferString(
					`{"choices":[{"message":{"role":"assistant","content":"Today: Light jacket\nOuterwear: Light rain shell\nFootwear: Waterproof sneakers\nExtras: Small umbrella\nWhy: Cool with a chance of showers.\n\nPoem:\nGray clouds drift low\nRain taps at the curb\nYour jacket keeps the chill in check\nWaterproof shoes carry you home"}}]}`,
				)),
			}, nil
		}),
	}

	profile, err := ClothingProfileByKey(DefaultClothingProfileKey)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	client := &LlamaClient{
		BaseURL:    "http://llama.local/v1/chat/completions",
		Model:      "Qwen/Qwen3-4B-GGUF",
		HTTPClient: clientHTTP,
	}
	weather := &Weather{
		ThreeDay: &ThreeDayWeather{
			Current: &WeatherPoint{Temperature: "51", Phrase: "Mostly cloudy"},
		},
	}

	got, err := client.RecommendClothesWithOptions(weather, profile, ClothingOptions{IncludePoem: true})
	if err != nil {
		t.Fatalf("recommend clothes with poem: %v", err)
	}
	if !strings.Contains(got, "Poem:") {
		t.Fatalf("expected poem in response: %q", got)
	}
	if !strings.Contains(captured.Messages[0].Content, "Poem: line followed by exactly four short lines") {
		t.Fatalf("missing poem guidance in system prompt: %q", captured.Messages[0].Content)
	}
	if !strings.Contains(captured.Messages[1].Content, "Also include a short poem about today's weather and the recommended clothes.") {
		t.Fatalf("missing poem guidance in user prompt: %q", captured.Messages[1].Content)
	}
}

func TestRecommendClothesWithPoemNormalizesMissingPoemLabel(t *testing.T) {
	clientHTTP := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(bytes.NewBufferString(
					`{"choices":[{"message":{"role":"assistant","content":"Today: Light jacket\nOuterwear: Windbreaker\nFootwear: Insulated boots\nExtras: Scarf or beanie\nWhy: Cool air and a sharp overnight drop call for layers.\n\nClouds drift over the cold,\nlayers keep the chill at bay,\nwool socks and boots hold fast,\nwinter's grip is soft in day."}}]}`,
				)),
			}, nil
		}),
	}

	profile, err := ClothingProfileByKey(DefaultClothingProfileKey)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	client := &LlamaClient{
		BaseURL:    "http://llama.local/v1/chat/completions",
		Model:      "Qwen/Qwen3-4B-GGUF",
		HTTPClient: clientHTTP,
	}
	weather := &Weather{
		ThreeDay: &ThreeDayWeather{
			Current: &WeatherPoint{Temperature: "37", Phrase: "Cloudy"},
		},
	}

	got, err := client.RecommendClothesWithOptions(weather, profile, ClothingOptions{IncludePoem: true})
	if err != nil {
		t.Fatalf("recommend clothes with normalized poem: %v", err)
	}
	if !strings.Contains(got, "\n\nPoem:\nClouds drift over the cold,") {
		t.Fatalf("expected normalized poem label, got: %q", got)
	}
}

func TestClothingPrompts(t *testing.T) {
	profile, err := ClothingProfileByKey(DefaultClothingProfileKey)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	profile.Activity = "walking the dog"
	profile.ExtraNotes = "prefers waterproof shoes"

	systemPrompt, userPrompt := ClothingPrompts(&Weather{
		ThreeDay: &ThreeDayWeather{
			Current: &WeatherPoint{Temperature: "44", Phrase: "Rainy", PrecipChance: "80"},
		},
	}, profile)

	if !strings.Contains(systemPrompt, "Return exactly these lines in order: Today:, Outerwear:, Footwear:, Extras:, Why:.") {
		t.Fatalf("missing structured guidance in system prompt: %q", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "Make Why the most detailed line") {
		t.Fatalf("missing Why detail guidance in system prompt: %q", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "prefers waterproof shoes") {
		t.Fatalf("missing notes in system prompt: %q", systemPrompt)
	}
	if !strings.Contains(userPrompt, "Expected activity: walking the dog.") {
		t.Fatalf("missing activity in user prompt: %q", userPrompt)
	}
	if !strings.Contains(userPrompt, "Current: 44F, Rainy with 80% precip chance.") {
		t.Fatalf("missing weather summary in user prompt: %q", userPrompt)
	}
}

func TestClothingPromptsWithPoem(t *testing.T) {
	profile, err := ClothingProfileByKey(DefaultClothingProfileKey)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}

	systemPrompt, userPrompt := ClothingPromptsWithOptions(&Weather{
		ThreeDay: &ThreeDayWeather{
			Current: &WeatherPoint{Temperature: "44", Phrase: "Rainy", PrecipChance: "80"},
		},
	}, profile, ClothingOptions{IncludePoem: true})

	if !strings.Contains(systemPrompt, "Poem: line followed by exactly four short lines") {
		t.Fatalf("missing poem guidance in system prompt: %q", systemPrompt)
	}
	if !strings.Contains(userPrompt, "Also include a short poem about today's weather and the recommended clothes.") {
		t.Fatalf("missing poem guidance in user prompt: %q", userPrompt)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
