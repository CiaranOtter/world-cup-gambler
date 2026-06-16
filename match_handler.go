// handlers_matches.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
)

// cache is the package-level data cache populated from worldcup26.ir.
// Initialized in main().
var cache *DataCache

// matchesHandler handles GET /api/matches
// Optional query params: ?group=A, ?matchday=3, ?finished=true
func matchesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	matches := cache.Matches()

	if group := r.URL.Query().Get("group"); group != "" {
		matches = filterMatches(matches, func(m Match) bool {
			return strings.EqualFold(m.Group, group)
		})
	}

	if matchday := r.URL.Query().Get("matchday"); matchday != "" {
		matches = filterMatches(matches, func(m Match) bool {
			return m.Matchday == matchday
		})
	}

	if finished := r.URL.Query().Get("finished"); finished != "" {
		want := finished == "true"
		matches = filterMatches(matches, func(m Match) bool {
			return m.Finished == want
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Matchday != matches[j].Matchday {
			return matches[i].Matchday < matches[j].Matchday
		}
		return matches[i].LocalDate < matches[j].LocalDate
	})

	writeJSON(w, http.StatusOK, matches)
}

// matchByIDHandler handles GET /api/matches/{id}
func matchByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/matches/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	match, ok := cache.MatchByID(id)
	if !ok {
		http.Error(w, "match not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, match)
}

// teamsHandler handles GET /api/teams
func teamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	teamsByID := cache.Teams()
	teams := make([]TeamInfo, 0, len(teamsByID))
	for _, t := range teamsByID {
		teams = append(teams, t)
	}

	if group := r.URL.Query().Get("group"); group != "" {
		filtered := make([]TeamInfo, 0, len(teams))
		for _, t := range teams {
			if strings.EqualFold(t.Groups, group) {
				filtered = append(filtered, t)
			}
		}
		teams = filtered
	}

	sort.Slice(teams, func(i, j int) bool {
		return teams[i].NameEN < teams[j].NameEN
	})

	writeJSON(w, http.StatusOK, teams)
}

// stadiumsHandler handles GET /api/stadiums
func stadiumsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stadiumsByID := cache.Stadiums()
	stadiums := make([]Stadium, 0, len(stadiumsByID))
	for _, s := range stadiumsByID {
		stadiums = append(stadiums, s)
	}

	sort.Slice(stadiums, func(i, j int) bool {
		return stadiums[i].NameEN < stadiums[j].NameEN
	})

	writeJSON(w, http.StatusOK, stadiums)
}

// groupsHandler handles GET /api/groups
func groupsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groups := cache.Groups()

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Group < groups[j].Group
	})

	writeJSON(w, http.StatusOK, groups)
}

// statusHandler handles GET /api/status — reports cache freshness,
// useful for debugging upstream connectivity.
func statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lastUpdated, err := cache.Status()

	resp := map[string]interface{}{
		"last_updated": lastUpdated,
		"match_count":  len(cache.Matches()),
	}
	if err != nil {
		resp["last_error"] = err.Error()
	}

	writeJSON(w, http.StatusOK, resp)
}

func filterMatches(matches []Match, keep func(Match) bool) []Match {
	out := make([]Match, 0, len(matches))
	for _, m := range matches {
		if keep(m) {
			out = append(out, m)
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON error: %v", err)
	}
}
