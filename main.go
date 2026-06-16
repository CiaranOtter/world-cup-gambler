// main.go
package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	var err error
	userStore, err = NewUserStore()
	if err != nil {
		log.Fatalf("initialising user store: %v", err)
	}

	client := NewWorldCupClient()
	cache = NewDataCache(client)
	cache.Start(30 * time.Second)

	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Pages
	mux.HandleFunc("/", indexHandler)

	// Match API
	mux.HandleFunc("/api/matches", matchesHandler)
	mux.HandleFunc("/api/matches/", matchByIDHandler)
	mux.HandleFunc("/api/teams", teamsHandler)
	mux.HandleFunc("/api/stadiums", stadiumsHandler)
	mux.HandleFunc("/api/groups", groupsHandler)
	mux.HandleFunc("/api/status", statusHandler)

	// User / game API
	mux.HandleFunc("/api/users", usersHandler)
	mux.HandleFunc("/api/users/", userByIDHandler)
	mux.HandleFunc("/api/leaderboard", leaderboardHandler)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "./static/index.html")
}
