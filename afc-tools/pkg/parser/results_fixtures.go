package parser

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/afc-tools/pkg/models"
)

var (
	titleRe = regexp.MustCompile(`(?is)<title>\s*([^<]+?)\s*</title>`)
	tableRe = regexp.MustCompile(`(?is)<table class="cols-0">\s*<caption>\s*(.*?)\s*</caption>\s*<tbody>\s*(.*?)\s*</tbody>\s*</table>`)
	rowRe   = regexp.MustCompile(`(?is)<tr>\s*(.*?)\s*</tr>`)

	timeRe        = regexp.MustCompile(`(?is)<time[^>]*datetime="([^"]+)"[^>]*>(.*?)</time>`)
	homeTeamRe    = regexp.MustCompile(`(?is)<td class="views-field views-field-nothing-1">\s*(.*?)\s*</td>`)
	scoreRe       = regexp.MustCompile(`(?is)<td class="final-score views-field views-field-nothing-1">\s*(.*?)\s*</td>`)
	awayTeamRe    = regexp.MustCompile(`(?is)<td class="views-field views-field-nothing-2">\s*(.*?)\s*</td>`)
	competitionRe = regexp.MustCompile(`(?is)<td class="views-field views-field-field-competition">\s*(.*?)\s*</td>`)
	scoreNumsRe   = regexp.MustCompile(`^\s*(\d+)\s*-\s*(\d+)\s*$`)
	tagRe         = regexp.MustCompile(`(?is)<[^>]+>`)
	spaceRe       = regexp.MustCompile(`\s+`)
	yearSuffixRe  = regexp.MustCompile(`(\d{4})\s*$`)
)

func ParseResultsAndFixturesList(data []byte) (*models.ResultsAndFixturesList, error) {
	content := string(data)

	titleMatch := titleRe.FindStringSubmatch(content)
	if len(titleMatch) < 2 {
		return nil, fmt.Errorf("parse title: no page title found")
	}

	out := &models.ResultsAndFixturesList{
		Title: cleanText(titleMatch[1]),
	}

	tables := tableRe.FindAllStringSubmatch(content, -1)
	if len(tables) == 0 {
		return nil, fmt.Errorf("parse fixture tables: no monthly tables found")
	}

	for _, table := range tables {
		monthLabel := cleanText(table[1])
		rows := rowRe.FindAllStringSubmatch(table[2], -1)
		for _, row := range rows {
			match, err := parseRow(monthLabel, row[1])
			if err != nil {
				return nil, err
			}
			out.Matches = append(out.Matches, match)
		}
	}

	return out, nil
}

func parseRow(monthLabel, rowHTML string) (models.Match, error) {
	timeMatch := timeRe.FindStringSubmatch(rowHTML)
	if len(timeMatch) < 3 {
		return models.Match{}, fmt.Errorf("parse row in %q: missing time element", monthLabel)
	}

	kickoffLabel := cleanText(timeMatch[2])
	kickoff, err := parseKickoff(monthLabel, timeMatch[1], kickoffLabel)
	if err != nil {
		return models.Match{}, fmt.Errorf("parse row in %q: parse kickoff %q: %w", monthLabel, timeMatch[1], err)
	}

	homeTeam, err := singleField(rowHTML, homeTeamRe, "home team", monthLabel)
	if err != nil {
		return models.Match{}, err
	}

	rawScoreText, err := singleField(rowHTML, scoreRe, "score", monthLabel)
	if err != nil {
		return models.Match{}, err
	}

	awayTeam, err := singleField(rowHTML, awayTeamRe, "away team", monthLabel)
	if err != nil {
		return models.Match{}, err
	}

	competition, err := singleField(rowHTML, competitionRe, "competition", monthLabel)
	if err != nil {
		return models.Match{}, err
	}

	match := models.Match{
		MonthLabel:    monthLabel,
		Kickoff:       kickoff,
		KickoffLabel:  kickoffLabel,
		HomeTeam:      homeTeam,
		AwayTeam:      awayTeam,
		Competition:   competition,
		RawScoreText:  rawScoreText,
		ArsenalIsHome: strings.EqualFold(homeTeam, "Arsenal"),
	}

	if scoreMatch := scoreNumsRe.FindStringSubmatch(rawScoreText); len(scoreMatch) == 3 {
		homeScore, err := strconv.Atoi(scoreMatch[1])
		if err != nil {
			return models.Match{}, fmt.Errorf("parse row in %q: parse home score %q: %w", monthLabel, scoreMatch[1], err)
		}

		awayScore, err := strconv.Atoi(scoreMatch[2])
		if err != nil {
			return models.Match{}, fmt.Errorf("parse row in %q: parse away score %q: %w", monthLabel, scoreMatch[2], err)
		}

		match.Status = models.MatchStatusFinished
		match.HomeScore = intPtr(homeScore)
		match.AwayScore = intPtr(awayScore)
		return match, nil
	}

	match.Status = models.MatchStatusScheduled
	return match, nil
}

func singleField(rowHTML string, re *regexp.Regexp, fieldName, monthLabel string) (string, error) {
	match := re.FindStringSubmatch(rowHTML)
	if len(match) < 2 {
		return "", fmt.Errorf("parse row in %q: missing %s", monthLabel, fieldName)
	}

	return cleanText(match[1]), nil
}

func cleanText(raw string) string {
	withoutTags := tagRe.ReplaceAllString(raw, " ")
	unescaped := html.UnescapeString(withoutTags)
	return strings.TrimSpace(spaceRe.ReplaceAllString(unescaped, " "))
}

func parseKickoff(monthLabel, rawDateTime, kickoffLabel string) (time.Time, error) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		return time.Time{}, fmt.Errorf("load Europe/London timezone: %w", err)
	}

	yearMatch := yearSuffixRe.FindStringSubmatch(monthLabel)
	if len(yearMatch) != 2 {
		return time.Time{}, fmt.Errorf("extract year from month label %q", monthLabel)
	}

	localKickoff, err := time.ParseInLocation("Mon Jan 2 - 15:04 2006", kickoffLabel+" "+yearMatch[1], london)
	if err == nil {
		return localKickoff.UTC(), nil
	}

	parsed, parseErr := time.Parse(time.RFC3339, rawDateTime)
	if parseErr != nil {
		return time.Time{}, err
	}

	return parsed.UTC(), nil
}

func intPtr(v int) *int {
	return &v
}
