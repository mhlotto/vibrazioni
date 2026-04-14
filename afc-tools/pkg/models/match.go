package models

import "time"

type MatchStatus string

const (
	MatchStatusScheduled MatchStatus = "scheduled"
	MatchStatusFinished  MatchStatus = "finished"
)

type Match struct {
	MonthLabel    string      `json:"month_label"`
	Kickoff       time.Time   `json:"kickoff"`
	KickoffLabel  string      `json:"kickoff_label"`
	HomeTeam      string      `json:"home_team"`
	AwayTeam      string      `json:"away_team"`
	Competition   string      `json:"competition"`
	Status        MatchStatus `json:"status"`
	RawScoreText  string      `json:"raw_score_text"`
	ArsenalIsHome bool        `json:"arsenal_is_home"`
	HomeScore     *int        `json:"home_score,omitempty"`
	AwayScore     *int        `json:"away_score,omitempty"`
}

type ResultsAndFixturesList struct {
	Title   string  `json:"title"`
	Matches []Match `json:"matches"`
}
