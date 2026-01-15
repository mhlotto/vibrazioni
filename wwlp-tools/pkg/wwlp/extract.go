package wwlp

import (
	"fmt"
	"sort"
)

func ArticleTitle(a Article) string {
	if a.HomePageTitle != "" {
		return a.HomePageTitle
	}
	return a.Title
}

func GetArticles(tv *TemplateVars, source string, listIndex int) ([]Article, error) {
	switch source {
	case "top":
		if tv.TopStories == nil {
			return nil, fmt.Errorf("top_stories missing")
		}
		return tv.TopStories.Articles, nil
	case "additional":
		if tv.AdditionalTopStories == nil {
			return nil, fmt.Errorf("additional_top_stories missing")
		}
		return tv.AdditionalTopStories.Articles, nil
	case "headline":
		if listIndex < 0 || listIndex >= len(tv.HeadlineLists) {
			return nil, fmt.Errorf("headline list index out of range")
		}
		return tv.HeadlineLists[listIndex].Articles, nil
	default:
		return nil, fmt.Errorf("unknown source: %s", source)
	}
}

func HeadlineListTitles(tv *TemplateVars) []string {
	out := make([]string, 0, len(tv.HeadlineLists))
	for i, hl := range tv.HeadlineLists {
		label := fmt.Sprintf("%d: %s", i, hl.Title)
		if hl.Provider != "" {
			label = fmt.Sprintf("%s (%s)", label, hl.Provider)
		}
		out = append(out, label)
	}
	return out
}

func AlertTypes(tv *TemplateVars) []string {
	if tv.AlertBanners == nil {
		return nil
	}
	out := make([]string, 0, len(tv.AlertBanners.Messages))
	for k := range tv.AlertBanners.Messages {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func AlertsByType(tv *TemplateVars, alertType string) ([]AlertMessage, error) {
	if tv.AlertBanners == nil {
		return nil, fmt.Errorf("alert_banners missing")
	}
	msgs, ok := tv.AlertBanners.Messages[alertType]
	if !ok {
		return nil, fmt.Errorf("alert type not found: %s", alertType)
	}
	return msgs, nil
}
