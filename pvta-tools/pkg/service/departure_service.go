package service

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/client"
	"github.com/mhlotto/vibrazioni/pvta-tools/pkg/models"
)

var htmlTagRe = regexp.MustCompile(`(?s)<[^>]*>`)

type DepartureService struct {
	client *client.Client
}

func NewDepartureService(c *client.Client) *DepartureService {
	return &DepartureService{client: c}
}

func (s *DepartureService) GetBoardForStop(ctx context.Context, stopID int) (*models.DepartureBoard, error) {
	text, err := s.client.GetText(ctx, fmt.Sprintf("/Minimal/Departures/ForStop?stopId=%d", stopID))
	if err != nil {
		return nil, err
	}
	board, err := parseDepartureBoard(text)
	if err != nil {
		return nil, err
	}
	return board, nil
}

func parseDepartureBoard(input string) (*models.DepartureBoard, error) {
	lines := normalizeHTMLText(input)
	board := &models.DepartureBoard{}
	var current *models.DepartureGroup
	inBoard := false

	for _, line := range lines {
		switch {
		case strings.EqualFold(line, "Upcoming Estimated Departures from"):
			inBoard = true
			current = nil
			continue
		case !inBoard:
			continue
		case strings.HasPrefix(line, "Last updated on:"):
			board.LastUpdated = strings.TrimSpace(strings.TrimPrefix(line, "Last updated on:"))
			current = nil
		case board.StopName == "":
			board.StopName = line
		case looksLikeDepartureTime(line):
			if current != nil {
				current.Times = append(current.Times, line)
			}
		default:
			board.Groups = append(board.Groups, models.DepartureGroup{
				RouteAndDirection: line,
			})
			current = &board.Groups[len(board.Groups)-1]
		}
	}

	if board.StopName == "" {
		return nil, fmt.Errorf("unable to parse departure board")
	}
	return board, nil
}

func normalizeHTMLText(input string) []string {
	text := html.UnescapeString(input)
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = htmlTagRe.ReplaceAllString(text, "\n")
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Copyright ") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func looksLikeDepartureTime(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return false
	}
	return strings.HasSuffix(s, "AM") || strings.HasSuffix(s, "PM")
}
