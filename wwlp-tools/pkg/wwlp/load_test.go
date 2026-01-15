package wwlp

import "testing"

func TestValidateTemplateVarsShapeOK(t *testing.T) {
	input := []byte(`{
  "top_stories": {},
  "additional_top_stories": {},
  "headline_lists": [],
  "weather": {},
  "alert_banners": {}
}`)
	warnings := ValidateTemplateVarsShape(input)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestValidateTemplateVarsShapeMissingAndWrongType(t *testing.T) {
	input := []byte(`{
  "top_stories": [],
  "headline_lists": {},
  "weather": {},
  "alert_banners": {}
}`)
	warnings := ValidateTemplateVarsShape(input)
	if len(warnings) == 0 {
		t.Fatalf("expected warnings, got none")
	}
	expected := map[string]bool{
		"missing key: additional_top_stories":    false,
		"unexpected type for top_stories: array": false,
		"unexpected type for headline_lists: object": false,
	}
	for _, w := range warnings {
		if _, ok := expected[w]; ok {
			expected[w] = true
		}
	}
	for key, found := range expected {
		if !found {
			t.Fatalf("missing expected warning: %s", key)
		}
	}
}
