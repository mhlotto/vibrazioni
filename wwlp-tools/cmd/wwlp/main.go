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

Default input is fetched from https://www.wwlp.com/wp-json/lakana/v1/template-variables/
Use --file for a saved JSON file.
`
	fmt.Fprint(os.Stderr, msg)
}

const defaultTemplateVarsURL = "https://www.wwlp.com/wp-json/lakana/v1/template-variables/"

func isStdinPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}

func loadTemplateVarsFromArgs(file string) *wwlp.TemplateVars {
	var (
		tv  *wwlp.TemplateVars
		err error
	)
	switch {
	case file != "":
		if file == "-" {
			tv, err = wwlp.LoadTemplateVars(os.Stdin)
		} else {
			tv, err = wwlp.LoadTemplateVarsFile(file)
		}
	case isStdinPiped():
		tv, err = wwlp.LoadTemplateVars(os.Stdin)
	default:
		tv, err = wwlp.LoadTemplateVarsURL(defaultTemplateVarsURL)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return tv
}

func headlines(args []string) {
	fs := flag.NewFlagSet("headlines", flag.ExitOnError)
	file := fs.String("file", "", "Input JSON file (or - for stdin)")
	source := fs.String("source", "top", "Source: top, additional, headline")
	listIndex := fs.Int("list", 0, "Headline list index (for source=headline)")
	limit := fs.Int("limit", 0, "Max items (0 means all)")
	fs.Parse(args)

	tv := loadTemplateVarsFromArgs(*file)
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
	fs.Parse(args)

	tv := loadTemplateVarsFromArgs(*file)
	for _, line := range wwlp.HeadlineListTitles(tv) {
		fmt.Println(line)
	}
}

func weather(args []string) {
	fs := flag.NewFlagSet("weather", flag.ExitOnError)
	file := fs.String("file", "", "Input JSON file (or - for stdin)")
	mode := fs.String("mode", "three-day", "Mode: current, three-day, hourly, seven-day")
	limit := fs.Int("limit", 0, "Max items for hourly or seven-day")
	fs.Parse(args)

	tv := loadTemplateVarsFromArgs(*file)
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
		printSevenDay(tv.Weather.SevenDay, *limit)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown mode: %s\n", *mode)
		os.Exit(1)
	}
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

func printSevenDay(items []wwlp.DailyForecast, limit int) {
	for i, d := range items {
		if limit > 0 && i >= limit {
			break
		}
		precip := formatPrecip(d.PrecipChance)
		fmt.Printf("%s: %sF/%sF %s%s\n", d.DayOfWeek, d.MaxTemperature, d.MinTemperature, d.ShortPhrase, precip)
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
	alertType := fs.String("type", "", "Alert type name")
	listTypes := fs.Bool("list-types", false, "List available alert types")
	fs.Parse(args)

	tv := loadTemplateVarsFromArgs(*file)

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
