package wwlp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultClothingProfileKey = "mid40-male"

type ClothingProfile struct {
	Key         string
	DisplayName string
	PromptHint  string
	Activity    string
	ExtraNotes  string
}

type ClothingOptions struct {
	IncludePoem bool
	SassyTone   bool
}

var clothingProfiles = []ClothingProfile{
	{
		Key:         DefaultClothingProfileKey,
		DisplayName: "Mid-40s Male",
		PromptHint:  "The user is a mid-40s man. Default to practical, age-appropriate everyday clothing for errands, commuting, and normal outdoor activity.",
	},
	{
		Key:         "mid40-female",
		DisplayName: "Mid-40s Female",
		PromptHint:  "The user is a mid-40s woman. Default to practical, age-appropriate everyday clothing for errands, commuting, and normal outdoor activity.",
	},
	{
		Key:         "teen",
		DisplayName: "Teenager",
		PromptHint:  "The user is a teenager. Keep the advice casual, flexible, and school-appropriate without sounding childish.",
	},
	{
		Key:         "young-adult-male",
		DisplayName: "Young Adult Male",
		PromptHint:  "The user is a young adult man. Keep the advice practical, comfortable, and suitable for everyday plans.",
	},
	{
		Key:         "young-adult-female",
		DisplayName: "Young Adult Female",
		PromptHint:  "The user is a young adult woman. Keep the advice practical, comfortable, and suitable for everyday plans.",
	},
	{
		Key:         "senior",
		DisplayName: "Senior Adult",
		PromptHint:  "The user is a senior adult. Prioritize warmth, comfort, stable footwear, and weather protection.",
	},
}

type LlamaClient struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

type chatCompletionsRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func ClothingProfileKeys() []string {
	out := make([]string, 0, len(clothingProfiles))
	for _, p := range clothingProfiles {
		out = append(out, p.Key)
	}
	return out
}

func ClothingProfileByKey(key string) (ClothingProfile, error) {
	key = strings.TrimSpace(strings.ToLower(key))
	for _, p := range clothingProfiles {
		if p.Key == key {
			return p, nil
		}
	}
	return ClothingProfile{}, fmt.Errorf("unknown profile: %s", key)
}

func NewLlamaClient(host string, port int, model string) *LlamaClient {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	host = strings.TrimRight(host, "/")
	baseURL := fmt.Sprintf("%s:%d/v1/chat/completions", host, port)
	return &LlamaClient{
		BaseURL: baseURL,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (c *LlamaClient) RecommendClothes(weather *Weather, profile ClothingProfile) (string, error) {
	return c.RecommendClothesWithOptions(weather, profile, ClothingOptions{})
}

func (c *LlamaClient) RecommendClothesWithOptions(weather *Weather, profile ClothingProfile, options ClothingOptions) (string, error) {
	if weather == nil {
		return "", fmt.Errorf("weather missing")
	}
	systemPrompt, userPrompt := ClothingPromptsWithOptions(weather, profile, options)
	reqBody := chatCompletionsRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		Temperature: 0.4,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("chat completion request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("chat completion status: %s", resp.Status)
	}

	var out chatCompletionsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if out.Error != nil && strings.TrimSpace(out.Error.Message) != "" {
		return "", fmt.Errorf("chat completion error: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("chat completion returned no choices")
	}
	content := strings.TrimSpace(out.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("chat completion returned empty content")
	}
	if options.IncludePoem {
		content = normalizeClothingPoemOutput(content)
	}
	return content, nil
}

func ClothingPrompts(weather *Weather, profile ClothingProfile) (string, string) {
	return ClothingPromptsWithOptions(weather, profile, ClothingOptions{})
}

func ClothingPromptsWithOptions(weather *Weather, profile ClothingProfile, options ClothingOptions) (string, string) {
	return clothingSystemPrompt(profile, options), buildClothingPrompt(weather, profile, options)
}

func clothingSystemPrompt(profile ClothingProfile, options ClothingOptions) string {
	parts := []string{
		"You recommend what to wear based on a local weather forecast.",
		profile.PromptHint,
		"Be specific and practical.",
		"Use concise plain text.",
		"Return exactly these lines in order: Today:, Outerwear:, Footwear:, Extras:, Why:.",
		"Keep Today, Outerwear, Footwear, and Extras short and concrete.",
		"Make Why the most detailed line and explain the main weather reasons behind the recommendation.",
		"Do not mention being an AI or describe the prompt.",
	}
	if options.SassyTone {
		parts = append(parts, "Use a playful, catty, mildly sassy tone. Be funny without being cruel, obscene, or mean-spirited.")
	}
	if options.IncludePoem {
		parts = append(parts, "After the Why line, add a blank line, then a Poem: line followed by exactly four short lines about the weather and the outfit.")
	}
	if strings.TrimSpace(profile.ExtraNotes) != "" {
		parts = append(parts, "Additional wearer details: "+strings.TrimSpace(profile.ExtraNotes)+".")
	}
	return strings.Join(parts, " ")
}

func buildClothingPrompt(weather *Weather, profile ClothingProfile, options ClothingOptions) string {
	var parts []string
	parts = append(parts, "Give clothing advice for this profile: "+profile.DisplayName+".")
	if strings.TrimSpace(profile.Activity) != "" {
		parts = append(parts, "Expected activity: "+strings.TrimSpace(profile.Activity)+".")
	}
	if strings.TrimSpace(profile.ExtraNotes) != "" {
		parts = append(parts, "Extra wearer notes: "+strings.TrimSpace(profile.ExtraNotes)+".")
	}
	if options.SassyTone {
		parts = append(parts, "Make the advice sound stylishly judgmental in a fun way, while still being genuinely useful.")
	}
	if options.IncludePoem {
		parts = append(parts, "Also include a short poem about today's weather and the recommended clothes.")
	}
	parts = append(parts, "Weather summary:")

	if weather.ThreeDay != nil {
		if weather.ThreeDay.Current != nil {
			parts = append(parts, fmt.Sprintf(
				"Current: %sF, %s%s.",
				strings.TrimSpace(weather.ThreeDay.Current.Temperature),
				strings.TrimSpace(weather.ThreeDay.Current.Phrase),
				formatPromptPrecip(weather.ThreeDay.Current.PrecipChance),
			))
		}
		if weather.ThreeDay.Tonight != nil {
			parts = append(parts, fmt.Sprintf(
				"Tonight: %sF, %s%s.",
				strings.TrimSpace(weather.ThreeDay.Tonight.Temperature),
				strings.TrimSpace(weather.ThreeDay.Tonight.Phrase),
				formatPromptPrecip(weather.ThreeDay.Tonight.PrecipChance),
			))
		}
		if weather.ThreeDay.Tomorrow != nil {
			parts = append(parts, fmt.Sprintf(
				"Tomorrow: %sF, %s%s.",
				strings.TrimSpace(weather.ThreeDay.Tomorrow.Temperature),
				strings.TrimSpace(weather.ThreeDay.Tomorrow.Phrase),
				formatPromptPrecip(weather.ThreeDay.Tomorrow.PrecipChance),
			))
		}
	}

	if len(weather.SevenDay) > 0 {
		today := weather.SevenDay[0]
		line := fmt.Sprintf(
			"Daily outlook: %s high %sF low %sF, %s%s.",
			strings.TrimSpace(today.DayOfWeek),
			strings.TrimSpace(today.MaxTemperature),
			strings.TrimSpace(today.MinTemperature),
			strings.TrimSpace(today.ShortPhrase),
			formatPromptPrecip(today.PrecipChance),
		)
		parts = append(parts, line)
		if strings.TrimSpace(today.DayNarrative) != "" {
			parts = append(parts, "Day narrative: "+strings.TrimSpace(today.DayNarrative)+".")
		}
		if strings.TrimSpace(today.NightNarrative) != "" {
			parts = append(parts, "Night narrative: "+strings.TrimSpace(today.NightNarrative)+".")
		}
	}

	if len(weather.Hourly) > 0 {
		hourly := weather.Hourly
		if len(hourly) > 4 {
			hourly = hourly[:4]
		}
		var items []string
		for _, h := range hourly {
			items = append(items, fmt.Sprintf("%s %sF %s", strings.TrimSpace(h.Time), strings.TrimSpace(h.Temperature), strings.TrimSpace(h.LongPhrase)))
		}
		parts = append(parts, "Near-term hourly: "+strings.Join(items, "; ")+".")
	}

	parts = append(parts, "Focus on what the person should wear outside today, not indoor fashion.")
	return strings.Join(parts, "\n")
}

func formatPromptPrecip(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "0" {
		return ""
	}
	return fmt.Sprintf(" with %s%% precip chance", p)
}

func normalizeClothingPoemOutput(content string) string {
	if strings.Contains(content, "\nPoem:") || strings.HasPrefix(content, "Poem:") {
		return content
	}

	lines := strings.Split(content, "\n")
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed = append(trimmed, strings.TrimRight(line, " \t"))
	}

	if len(trimmed) < 9 {
		return content
	}
	if strings.TrimSpace(trimmed[0]) == "" {
		return content
	}
	requiredPrefixes := []string{"Today:", "Outerwear:", "Footwear:", "Extras:", "Why:"}
	for i, prefix := range requiredPrefixes {
		if i >= len(trimmed) || !strings.HasPrefix(strings.TrimSpace(trimmed[i]), prefix) {
			return content
		}
	}

	start := len(trimmed) - 4
	for _, line := range trimmed[start:] {
		if strings.TrimSpace(line) == "" {
			return content
		}
	}

	head := strings.Join(trimmed[:start], "\n")
	poem := strings.Join(trimmed[start:], "\n")
	return strings.TrimRight(head, "\n") + "\n\nPoem:\n" + poem
}
