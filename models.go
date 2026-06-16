// models.go
package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// flexString unmarshals a JSON value that may arrive as either a string
// or a number into a Go string. The worldcup26.ir API is inconsistent
// about this across endpoints (e.g. "matchday": "1" vs matchday: 1).
type flexString string

func (f *flexString) UnmarshalJSON(data []byte) error {
	// Treat null as empty.
	if string(data) == "null" {
		*f = ""
		return nil
	}

	// Try string first.
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = flexString(s)
		return nil
	}

	// Fall back to number.
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = flexString(n.String())
		return nil
	}

	return fmt.Errorf("flexString: cannot unmarshal %s", string(data))
}

func (f flexString) String() string {
	return string(f)
}

// flexBool unmarshals a JSON value that may arrive as either a bool or
// a string ("true"/"false"/"TRUE") into a Go bool.
type flexBool bool

func (f *flexBool) UnmarshalJSON(data []byte) error {
	// Treat null/missing as false.
	if string(data) == "null" {
		*f = false
		return nil
	}

	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*f = flexBool(b)
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		parsed, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("flexBool: cannot parse %q as bool", s)
		}
		*f = flexBool(parsed)
		return nil
	}

	return fmt.Errorf("flexBool: cannot unmarshal %s", string(data))
}

// scorerList unmarshals the home_scorers/away_scorers fields, which may
// arrive as:
//   - a real JSON array: ["J. Quiñones 9'", "R. Jiménez 67'"]
//   - null
//   - a Postgres array literal serialized as a JSON string, e.g.
//     "{\"J. Quiñones 9'\",\"R. Jiménez 67'\"}" — note this may use
//     curly/smart quotes (\u201c \u201d) instead of straight quotes.
type scorerList []string

func (s *scorerList) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = nil
		return nil
	}

	// Try a real JSON array first.
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = scorerList(arr)
		return nil
	}

	// Fall back: a JSON string containing a Postgres array literal.
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("scorerList: cannot unmarshal %s", string(data))
	}

	*s = scorerList(parsePostgresArrayLiteral(str))
	return nil
}

// parsePostgresArrayLiteral parses strings like
// `{"J. Quiñones 9'","R. Jiménez 67'"}` (including variants using
// curly/smart quotes “ ”) into a slice of unquoted entries. Returns
// nil for empty or unparseable input.
func parsePostgresArrayLiteral(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" || strings.EqualFold(s, "null") {
		return nil
	}

	// Strip surrounding braces.
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	if s == "" {
		return nil
	}

	// Split on commas separating quoted entries.
	parts := strings.Split(s, ",")

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// Trim straight and curly/smart quotes from both ends.
		p = strings.Trim(p, `"`)
		p = strings.Trim(p, "\u201c\u201d") // “ ”
		p = strings.Trim(p, "\u2018\u2019") // ‘ ’
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// Team represents a team's basic info as embedded in a Match.
type Team struct {
	ID     string `json:"id"`
	NameEN string `json:"name_en"`
	NameFA string `json:"name_fa"`
}

// TeamInfo represents the full team record from /get/teams.
type TeamInfo struct {
	ID       flexString `json:"id"`
	NameEN   string     `json:"name_en"`
	NameFA   string     `json:"name_fa"`
	FifaCode string     `json:"fifa_code"`
	Groups   string     `json:"groups"`
	Flag     string     `json:"flag"`
}

// Stadium represents a stadium record from /get/stadiums.
type Stadium struct {
	ID        flexString `json:"id"`
	NameEN    string     `json:"name_en"`
	NameFA    string     `json:"name_fa"`
	FifaName  string     `json:"fifa_name"`
	CityEN    string     `json:"city_en"`
	CountryEN string     `json:"country_en"`
	Capacity  int        `json:"capacity"`
}

// GroupStanding represents a single team's row in a group table.
type GroupStanding struct {
	TeamID flexString `json:"team_id"`
	Points flexString `json:"pts"`
	GoalsF flexString `json:"gf"`
	GoalsA flexString `json:"ga"`
}

// GroupTable represents one group's standings from /get/groups.
type GroupTable struct {
	Group string          `json:"group"`
	Teams []GroupStanding `json:"teams"`
}

// Match represents a single game/result, with team info resolved
// from a separate teams lookup.
type Match struct {
	ID          string   `json:"id"`
	HomeScore   string   `json:"home_score"`
	AwayScore   string   `json:"away_score"`
	HomeScorers []string `json:"home_scorers"`
	AwayScorers []string `json:"away_scorers"`
	Group       string   `json:"group"`
	Matchday    string   `json:"matchday"`
	LocalDate   string   `json:"local_date"`
	PersianDate string   `json:"persian_date,omitempty"`
	Finished    bool     `json:"finished"`
	TimeElapsed string   `json:"time_elapsed"`
	Type        string   `json:"type"`

	StadiumID string  `json:"stadium_id"`
	HomeTeam  Team    `json:"home_team"`
	AwayTeam  Team    `json:"away_team"`
	Stadium   Stadium `json:"stadium,omitempty"`
}

// rawMatch mirrors the actual JSON shape returned by /get/games.
// Several fields are inconsistently typed (string vs number) across
// the API, so flexString/flexBool absorb that variance.
type rawMatch struct {
	ID          flexString `json:"id"`
	HomeTeamID  flexString `json:"home_team_id"`
	AwayTeamID  flexString `json:"away_team_id"`
	HomeScore   flexString `json:"home_score"`
	AwayScore   flexString `json:"away_score"`
	HomeScorers scorerList `json:"home_scorers"`
	AwayScorers scorerList `json:"away_scorers"`
	Group       string     `json:"group"`
	Matchday    flexString `json:"matchday"`
	LocalDate   string     `json:"local_date"`
	PersianDate string     `json:"persian_date"`
	StadiumID   flexString `json:"stadium_id"`
	Finished    flexBool   `json:"finished"`
	TimeElapsed string     `json:"time_elapsed"`
	Type        string     `json:"type"`

	// Some responses may embed team names directly; if present, they
	// take precedence over a teams-lookup join.
	HomeTeamNameEN string `json:"home_team_name_en"`
	HomeTeamNameFA string `json:"home_team_name_fa"`
	AwayTeamNameEN string `json:"away_team_name_en"`
	AwayTeamNameFA string `json:"away_team_name_fa"`
}

// ToMatch converts a rawMatch into the cleaner Match struct. Team
// names are resolved via the provided teams lookup (by team ID),
// falling back to any inline names present on the raw match.
func (r rawMatch) ToMatch(teams map[string]TeamInfo) Match {
	m := Match{
		ID:          r.ID.String(),
		HomeScore:   r.HomeScore.String(),
		AwayScore:   r.AwayScore.String(),
		HomeScorers: r.HomeScorers,
		AwayScorers: r.AwayScorers,
		Group:       r.Group,
		Matchday:    r.Matchday.String(),
		LocalDate:   r.LocalDate,
		PersianDate: r.PersianDate,
		Finished:    bool(r.Finished),
		TimeElapsed: r.TimeElapsed,
		Type:        r.Type,
		StadiumID:   r.StadiumID.String(),
	}

	m.HomeTeam = resolveTeam(r.HomeTeamID.String(), r.HomeTeamNameEN, r.HomeTeamNameFA, teams)
	m.AwayTeam = resolveTeam(r.AwayTeamID.String(), r.AwayTeamNameEN, r.AwayTeamNameFA, teams)

	return m
}

// resolveTeam looks up a team by ID in the teams map. If not found,
// it falls back to any inline name fields provided on the match.
func resolveTeam(id, fallbackEN, fallbackFA string, teams map[string]TeamInfo) Team {
	if info, ok := teams[id]; ok {
		return Team{
			ID:     id,
			NameEN: info.NameEN,
			NameFA: info.NameFA,
		}
	}

	return Team{
		ID:     id,
		NameEN: fallbackEN,
		NameFA: fallbackFA,
	}
}
