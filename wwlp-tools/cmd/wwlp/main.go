package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"wwlp-tools/pkg/wwlp"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "headlines":
		headlines(os.Args[2:])
	case "headline-lists":
		headlineLists(os.Args[2:])
	case "weather":
		weather(os.Args[2:])
	case "alerts":
		alerts(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	msg := `Usage: wwlp <command> [options]

Commands:
  headlines       List headlines with links
  headline-lists  List headline list titles and indexes
  weather         Show weather summaries
  alerts          Show alert messages

Default input is fetched from WWLP endpoints.
Use --file for saved JSON or HTML.
`
	fmt.Fprint(os.Stderr, msg)
}

const defaultTemplateVarsURL = "https://www.wwlp.com/wp-json/lakana/v1/template-variables/"
const defaultForecastDiscussionURL = "https://www.wwlp.com/weather/todays-forecast/forecast-discussion/"
const defaultWeatherAlertCounties = "25013,25015,25011,25003"

func isStdinPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}

func loadTemplateVarsFromArgs(file string, quiet bool) *wwlp.TemplateVars {
	var (
		tv       *wwlp.TemplateVars
		warnings []string
		err      error
	)
	switch {
	case file != "":
		if file == "-" {
			tv, warnings, err = wwlp.LoadTemplateVars(os.Stdin)
		} else {
			tv, warnings, err = wwlp.LoadTemplateVarsFile(file)
		}
	case isStdinPiped():
		tv, warnings, err = wwlp.LoadTemplateVars(os.Stdin)
	default:
		tv, warnings, err = wwlp.LoadTemplateVarsURL(defaultTemplateVarsURL)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !quiet {
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "Warning: JSON shape changed: %s\n", w)
		}
	}
	return tv
}

func headlines(args []string) {
	fs := flag.NewFlagSet("headlines", flag.ExitOnError)
	file := fs.String("file", "", "Input JSON file (or - for stdin)")
	quiet := fs.Bool("quiet-warning", false, "Suppress JSON shape warnings")
	source := fs.String("source", "top", "Source: top, additional, headline")
	listIndex := fs.Int("list", 0, "Headline list index (for source=headline)")
	limit := fs.Int("limit", 0, "Max items (0 means all)")
	fs.Parse(args)

	tv := loadTemplateVarsFromArgs(*file, *quiet)
	articles, err := wwlp.GetArticles(tv, *source, *listIndex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for i, a := range articles {
		if *limit > 0 && i >= *limit {
			break
		}
		title := wwlp.ArticleTitle(a)
		fmt.Printf("%s - %s\n", title, a.Link)
	}
}

func headlineLists(args []string) {
	fs := flag.NewFlagSet("headline-lists", flag.ExitOnError)
	file := fs.String("file", "", "Input JSON file (or - for stdin)")
	quiet := fs.Bool("quiet-warning", false, "Suppress JSON shape warnings")
	fs.Parse(args)

	tv := loadTemplateVarsFromArgs(*file, *quiet)
	for _, line := range wwlp.HeadlineListTitles(tv) {
		fmt.Println(line)
	}
}

func weather(args []string) {
	fs := flag.NewFlagSet("weather", flag.ExitOnError)
	file := fs.String("file", "", "Input file (JSON/HTML or - for stdin)")
	quiet := fs.Bool("quiet-warning", false, "Suppress JSON shape warnings")
	mode := fs.String("mode", "forecast", "Mode: forecast (default), current, three-day, hourly, seven-day, alerts")
	limit := fs.Int("limit", 0, "Max items for hourly or seven-day")
	short := fs.Bool("short", false, "Short output for seven-day")
	counties := fs.String("counties", defaultWeatherAlertCounties, "Counties list for weather alerts")
	fs.Parse(args)

	if *mode == "forecast" {
		discussion := loadForecastDiscussionFromArgs(*file)
		printForecastDiscussion(discussion)
		return
	}
	if *mode == "alerts" {
		alerts := loadWeatherAlertsFromArgs(*file, *counties)
		printWeatherAlerts(alerts)
		return
	}

	tv := loadTemplateVarsFromArgs(*file, *quiet)
	if tv.Weather == nil {
		fmt.Fprintln(os.Stderr, "error: weather missing")
		os.Exit(1)
	}

	switch *mode {
	case "current":
		if tv.Weather.ThreeDay == nil || tv.Weather.ThreeDay.Current == nil {
			fmt.Fprintln(os.Stderr, "error: current weather missing")
			os.Exit(1)
		}
		p := tv.Weather.ThreeDay.Current
		fmt.Printf("Current: %sF %s\n", p.Temperature, p.Phrase)
	case "three-day":
		if tv.Weather.ThreeDay == nil {
			fmt.Fprintln(os.Stderr, "error: three_day weather missing")
			os.Exit(1)
		}
		printThreeDay(tv.Weather.ThreeDay)
	case "hourly":
		printHourly(tv.Weather.Hourly, *limit)
	case "seven-day":
		printSevenDay(tv.Weather.SevenDay, *limit, *short)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}

func loadForecastDiscussionFromArgs(file string) *wwlp.ForecastDiscussion {
	var (
		article *wwlp.ForecastDiscussion
		err     error
	)
	switch {
	case file != "":
		if file == "-" {
			article, err = wwlp.LoadForecastDiscussion(os.Stdin)
		} else {
			article, err = wwlp.LoadForecastDiscussionFile(file)
		}
	default:
		article, err = wwlp.LoadForecastDiscussionURL(defaultForecastDiscussionURL)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return article
}

func printForecastDiscussion(article *wwlp.ForecastDiscussion) {
	if article.Headline != "" {
		fmt.Printf("Headline: %s\n", article.Headline)
	}
	if len(article.Authors) > 0 {
		fmt.Printf("Author: %s\n", strings.Join(article.Authors, ", "))
	}
	if article.DatePublished != "" {
		fmt.Printf("Published: %s\n", article.DatePublished)
	}
	if article.DateModified != "" {
		fmt.Printf("Modified: %s\n", article.DateModified)
	}
	if article.Headline != "" || len(article.Authors) > 0 || article.DatePublished != "" || article.DateModified != "" {
		fmt.Println()
	}
	if article.ArticleBody != "" {
		fmt.Println(wwlp.CleanForecastText(article.ArticleBody))
	}
}

func loadWeatherAlertsFromArgs(file, counties string) []wwlp.WeatherAlert {
	var (
		alerts []wwlp.WeatherAlert
		err    error
	)
	switch {
	case file != "":
		if file == "-" {
			alerts, err = wwlp.LoadWeatherAlerts(os.Stdin)
		} else {
			alerts, err = wwlp.LoadWeatherAlertsFile(file)
		}
	default:
		alerts, err = wwlp.LoadWeatherAlertsURL(counties)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return alerts
}

func printWeatherAlerts(alerts []wwlp.WeatherAlert) {
	if len(alerts) == 0 {
		fmt.Println("No active weather alerts.")
		return
	}
	for i, alert := range alerts {
		if i > 0 {
			fmt.Println()
		}
		payload, err := wwlp.ParseWeatherAlertPayload(alert.WeatherDetail.Payload)
		if err != nil {
			payload = nil
		}
		title := firstNonEmpty(
			alert.WeatherDetail.AlertType,
			alert.Description,
			payloadString(payload, func(p *wwlp.WeatherAlertPayload) string { return p.EventDescription }),
		)
		area := firstNonEmpty(
			alert.WeatherDetail.AreaName,
			alert.AreaName,
			payloadString(payload, func(p *wwlp.WeatherAlertPayload) string { return p.AreaName }),
		)
		severity := firstNonEmpty(
			alert.Severity,
			payloadString(payload, func(p *wwlp.WeatherAlertPayload) string { return p.Severity }),
		)
		if title != "" && area != "" {
			if severity != "" {
				fmt.Printf("%s - %s (%s)\n", title, area, severity)
			} else {
				fmt.Printf("%s - %s\n", title, area)
			}
		} else if title != "" {
			fmt.Println(title)
		}

		headline := firstNonEmpty(
			payloadString(payload, func(p *wwlp.WeatherAlertPayload) string { return p.HeadlineText }),
			payloadString(payload, func(p *wwlp.WeatherAlertPayload) string { return p.HeadlineTextAlt }),
		)
		if headline != "" {
			fmt.Printf("Headline: %s\n", headline)
		}

		effective := firstNonEmpty(
			payloadString(payload, func(p *wwlp.WeatherAlertPayload) string { return p.EffectiveTime }),
			alert.WeatherDetail.EffectiveTimestamp,
			alert.EffectiveTimestamp,
		)
		expires := firstNonEmpty(
			payloadString(payload, func(p *wwlp.WeatherAlertPayload) string { return p.ExpireTime }),
			payloadString(payload, func(p *wwlp.WeatherAlertPayload) string { return p.EndTime }),
			alert.WeatherDetail.ExpireTimestamp,
			alert.ExpireTimestamp,
		)
		if effective != "" || expires != "" {
			fmt.Printf("Effective: %s  Expires: %s\n", effective, expires)
		}

		desc := ""
		if payload != nil {
			desc = selectAlertDescription(payload)
		}
		if desc == "" {
			desc = strings.TrimSpace(alert.WeatherDetail.LongDescription)
		}
		if desc != "" {
			fmt.Println()
			fmt.Println(strings.TrimSpace(desc))
		}
	}
}

func selectAlertDescription(payload *wwlp.WeatherAlertPayload) string {
	for _, t := range payload.Texts {
		if strings.EqualFold(t.LanguageCode, "en-US") && t.Description != "" {
			return strings.TrimSpace(t.Description)
		}
	}
	for _, t := range payload.Texts {
		if t.Description != "" {
			return strings.TrimSpace(t.Description)
		}
	}
	return ""
}

func payloadString(payload *wwlp.WeatherAlertPayload, getter func(*wwlp.WeatherAlertPayload) string) string {
	if payload == nil {
		return ""
	}
	return getter(payload)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func printThreeDay(t *wwlp.ThreeDayWeather) {
	if t.Current != nil {
		fmt.Printf("Current: %sF %s\n", t.Current.Temperature, t.Current.Phrase)
	}
	if t.Tonight != nil {
		precip := formatPrecip(t.Tonight.PrecipChance)
		fmt.Printf("Tonight: %sF %s%s\n", t.Tonight.Temperature, t.Tonight.Phrase, precip)
	}
	if t.Tomorrow != nil {
		precip := formatPrecip(t.Tomorrow.PrecipChance)
		fmt.Printf("Tomorrow: %sF %s%s\n", t.Tomorrow.Temperature, t.Tomorrow.Phrase, precip)
	}
}

func printHourly(items []wwlp.HourlyForecast, limit int) {
	for i, h := range items {
		if limit > 0 && i >= limit {
			break
		}
		precip := formatPrecip(h.PrecipChance)
		fmt.Printf("%s %sF %s%s\n", h.Time, h.Temperature, h.LongPhrase, precip)
	}
}

func printSevenDay(items []wwlp.DailyForecast, limit int, short bool) {
	for i, d := range items {
		if limit > 0 && i >= limit {
			break
		}
		precip := formatPrecip(d.PrecipChance)
		if short || (d.DayNarrative == "" && d.NightNarrative == "") {
			fmt.Printf("%s: %sF/%sF %s%s\n", d.DayOfWeek, d.MaxTemperature, d.MinTemperature, d.ShortPhrase, precip)
			continue
		}
		fmt.Printf("%s: %sF/%sF %s%s\n", d.DayOfWeek, d.MaxTemperature, d.MinTemperature, d.ShortPhrase, precip)
		if d.DayNarrative != "" {
			fmt.Printf("  Day: %s\n", d.DayNarrative)
		}
		if d.NightNarrative != "" {
			fmt.Printf("  Night: %s\n", d.NightNarrative)
		}
	}
}

func formatPrecip(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "0" {
		return ""
	}
	return " (" + p + "% precip)"
}

func alerts(args []string) {
	fs := flag.NewFlagSet("alerts", flag.ExitOnError)
	file := fs.String("file", "", "Input JSON file (or - for stdin)")
	quiet := fs.Bool("quiet-warning", false, "Suppress JSON shape warnings")
	alertType := fs.String("type", "", "Alert type name")
	listTypes := fs.Bool("list-types", false, "List available alert types")
	weatherAlerts := fs.Bool("weather", false, "Fetch weather alerts from the weather service")
	counties := fs.String("counties", defaultWeatherAlertCounties, "Counties list for weather alerts")
	fs.Parse(args)

	if *weatherAlerts {
		alerts := loadWeatherAlertsFromArgs(*file, *counties)
		printWeatherAlerts(alerts)
		return
	}

	tv := loadTemplateVarsFromArgs(*file, *quiet)

	if *listTypes || *alertType == "" {
		for _, t := range wwlp.AlertTypes(tv) {
			fmt.Println(t)
		}
		if *alertType == "" {
			return
		}
	}

	msgs, err := wwlp.AlertsByType(tv, *alertType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	for _, m := range msgs {
		if m.URL != "" {
			fmt.Printf("%s - %s\n", m.Content, m.URL)
		} else {
			fmt.Println(m.Content)
		}
	}
}
