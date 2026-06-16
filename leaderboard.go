// leaderboard.go
package main

import (
	"sort"
)

// LeaderboardEntry is a user enriched with their team's current
// tournament stats derived from the live match cache.
//
// Score formula:
//
//	Points × 3  (group stage + knockout points)
//	+ Goals For × 1
//	+ Goal Difference × 1  (can be negative)
//	+ Knockout wins × 5    (matches of type != "group" that were won)
type LeaderboardEntry struct {
	UserProfile
	// Raw stats
	Played       int `json:"played"`
	Won          int `json:"won"`
	Drawn        int `json:"drawn"`
	Lost         int `json:"lost"`
	GoalsFor     int `json:"goals_for"`
	GoalsAgainst int `json:"goals_against"`
	GoalDiff     int `json:"goal_diff"`
	Points       int `json:"points"`
	KnockoutWins int `json:"knockout_wins"`
	// Composite game score
	Score int `json:"score"`
	Rank  int `json:"rank"`
}

// BuildLeaderboard computes a ranked leaderboard from the user list
// and the current match cache.
func BuildLeaderboard(users []*UserProfile, matches []Match) []LeaderboardEntry {
	entries := make([]LeaderboardEntry, 0, len(users))

	for _, u := range users {
		e := LeaderboardEntry{UserProfile: *u}

		for _, m := range matches {
			if !m.Finished {
				continue
			}

			homeGoals := parseInt(m.HomeScore)
			awayGoals := parseInt(m.AwayScore)
			isHome := m.HomeTeam.ID == u.TeamID
			isAway := m.AwayTeam.ID == u.TeamID

			if !isHome && !isAway {
				continue
			}

			e.Played++
			isKnockout := m.Type != "group"

			if isHome {
				e.GoalsFor += homeGoals
				e.GoalsAgainst += awayGoals
				switch {
				case homeGoals > awayGoals:
					e.Won++
					e.Points += 3
					if isKnockout {
						e.KnockoutWins++
					}
				case homeGoals == awayGoals:
					e.Drawn++
					e.Points++
				default:
					e.Lost++
				}
			} else {
				e.GoalsFor += awayGoals
				e.GoalsAgainst += homeGoals
				switch {
				case awayGoals > homeGoals:
					e.Won++
					e.Points += 3
					if isKnockout {
						e.KnockoutWins++
					}
				case awayGoals == homeGoals:
					e.Drawn++
					e.Points++
				default:
					e.Lost++
				}
			}
		}

		e.GoalDiff = e.GoalsFor - e.GoalsAgainst
		e.Score = (e.Points * 3) + e.GoalsFor + e.GoalDiff + (e.KnockoutWins * 5)
		entries = append(entries, e)
	}

	// Sort: score desc, goal diff desc, goals for desc, name asc.
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.GoalDiff != b.GoalDiff {
			return a.GoalDiff > b.GoalDiff
		}
		if a.GoalsFor != b.GoalsFor {
			return a.GoalsFor > b.GoalsFor
		}
		return a.Name < b.Name
	})

	for i := range entries {
		entries[i].Rank = i + 1
	}

	return entries
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
