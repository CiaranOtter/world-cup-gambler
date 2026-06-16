// handlers_users.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

var userStore *UserStore

// POST /api/users        — create a profile, get assigned a random team
// GET  /api/users        — list all users
// GET  /api/users/{id}   — get one user
// GET  /api/leaderboard  — ranked leaderboard with live team stats

func usersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listUsers(w, r)
	case http.MethodPost:
		createUser(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func userByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/users/")

	// POST /api/users/{id}/reroll
	if strings.HasSuffix(id, "/reroll") {
		id = strings.TrimSuffix(id, "/reroll")
		rerollUser(w, r, id)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if id == "" {
		http.NotFound(w, r)
		return
	}
	u, ok := userStore.GetByID(id)
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func rerollUser(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	teams := make([]TeamInfo, 0)
	for _, t := range cache.Teams() {
		teams = append(teams, t)
	}
	u, err := userStore.Reroll(id, teams)
	if err != nil {
		log.Printf("rerollUser: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userStore.All())
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if len(body.Name) > 32 {
		http.Error(w, "name too long (max 32 chars)", http.StatusBadRequest)
		return
	}

	teams := make([]TeamInfo, 0)
	for _, t := range cache.Teams() {
		teams = append(teams, t)
	}
	if len(teams) == 0 {
		http.Error(w, "team data not yet available, try again in a moment", http.StatusServiceUnavailable)
		return
	}

	u, err := userStore.Create(body.Name, teams)
	if err != nil {
		log.Printf("createUser: %v", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, u)
}

func leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entries := BuildLeaderboard(userStore.All(), cache.Matches())
	writeJSON(w, http.StatusOK, entries)
}
