package wwlp

type TemplateVars struct {
	TopStories          *StoryList     `json:"top_stories"`
	AdditionalTopStories *StoryList    `json:"additional_top_stories"`
	HeadlineLists       []HeadlineList `json:"headline_lists"`
	Weather             *Weather       `json:"weather"`
	AlertBanners        *AlertBanners  `json:"alert_banners"`
}

type StoryList struct {
	Title    string     `json:"title"`
	Articles []Article  `json:"articles"`
	ReadMore *ReadMore  `json:"read_more"`
}

type HeadlineList struct {
	Title     string    `json:"title"`
	Provider  string    `json:"provider"`
	Articles  []Article `json:"articles"`
	TitleLink string    `json:"title_link"`
}

type ReadMore struct {
	Title string `json:"title"`
	Link  string `json:"link"`
}

type Article struct {
	ID            int          `json:"id"`
	Title         string       `json:"title"`
	HomePageTitle string       `json:"home_page_title"`
	Link          string       `json:"link"`
	Category      *Category    `json:"category"`
	Date          *DateInfo    `json:"date"`
	MediaTypeIcon *MediaType   `json:"media_type_icon"`
	PostType      string       `json:"post_type"`
}

type Category struct {
	Title string `json:"title"`
	Link  string `json:"link"`
}

type DateInfo struct {
	Time     string `json:"time"`
	Datetime string `json:"datetime"`
}

type MediaType struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

type Weather struct {
	Hourly   []HourlyForecast `json:"hourly"`
	ThreeDay *ThreeDayWeather `json:"three_day"`
	SevenDay []DailyForecast  `json:"seven_day"`
}

type ThreeDayWeather struct {
	Current  *WeatherPoint `json:"current"`
	Tonight  *WeatherPoint `json:"tonight"`
	Tomorrow *WeatherPoint `json:"tomorrow"`
}

type WeatherPoint struct {
	Temperature  string `json:"temperature"`
	Icon         string `json:"icon"`
	Phrase       string `json:"phrase"`
	PrecipChance string `json:"precip_chance"`
}

type HourlyForecast struct {
	Time         string `json:"time"`
	Temperature  string `json:"temperature"`
	PrecipChance string `json:"precip_chance"`
	Humidity     string `json:"humidity"`
	Icon         string `json:"icon"`
	LongPhrase   string `json:"long_phrase"`
}

type DailyForecast struct {
	DayOfWeek     string `json:"day_of_week"`
	MaxTemperature string `json:"max_temperature"`
	MinTemperature string `json:"min_temperature"`
	PrecipChance  string `json:"precip_chance"`
	ShortPhrase   string `json:"short_phrase"`
	Time          string `json:"time"`
}

type AlertBanners struct {
	Messages map[string][]AlertMessage `json:"messages"`
}

type AlertMessage struct {
	Content string `json:"content"`
	URL     string `json:"url"`
	Start   any    `json:"start"`
	PostID  int    `json:"post_id"`
	UUID    string `json:"uuid"`
}
