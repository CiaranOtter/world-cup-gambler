// store.go
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

const storeFile = "./data/users.json"

// UserProfile represents a registered player and their assigned team.
type UserProfile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TeamID      string    `json:"team_id"`
	TeamName    string    `json:"team_name"`
	TeamFA      string    `json:"team_fa"`
	FifaCode    string    `json:"fifa_code"`
	HasRerolled bool      `json:"has_rerolled"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserStore is a thread-safe, file-backed store of user profiles.
type UserStore struct {
	mu    sync.RWMutex
	users map[string]*UserProfile // keyed by ID
}

func NewUserStore() (*UserStore, error) {
	s := &UserStore{users: make(map[string]*UserProfile)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Create registers a new user, assigns them a random team from those
// not already taken, and persists to disk.
func (s *UserStore) Create(name string, availableTeams []TeamInfo) (*UserProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Build set of taken team IDs.
	taken := make(map[string]bool, len(s.users))
	for _, u := range s.users {
		taken[u.TeamID] = true
	}

	// Filter to untaken teams.
	var pool []TeamInfo
	for _, t := range availableTeams {
		if !taken[t.ID.String()] {
			pool = append(pool, t)
		}
	}
	if len(pool) == 0 {
		return nil, fmt.Errorf("all teams have been assigned")
	}

	team := pool[rand.Intn(len(pool))]

	u := &UserProfile{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:      name,
		TeamID:    team.ID.String(),
		TeamName:  team.NameEN,
		TeamFA:    team.NameFA,
		FifaCode:  team.FifaCode,
		CreatedAt: time.Now(),
	}

	s.users[u.ID] = u

	if err := s.save(); err != nil {
		return nil, fmt.Errorf("saving user store: %w", err)
	}

	return u, nil
}

// All returns a copy of all users.
func (s *UserStore) All() []*UserProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*UserProfile, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	return out
}

// GetByID returns a single user.
func (s *UserStore) GetByID(id string) (*UserProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

// load reads persisted users from disk. Missing file is treated as
// empty store (first run).
func (s *UserStore) load() error {
	data, err := os.ReadFile(storeFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading user store: %w", err)
	}
	var list []*UserProfile
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("parsing user store: %w", err)
	}
	for _, u := range list {
		s.users[u.ID] = u
	}
	return nil
}

// Reroll assigns the user a new random team. Only allowed once.
func (s *UserStore) Reroll(id string, availableTeams []TeamInfo) (*UserProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	if u.HasRerolled {
		return nil, fmt.Errorf("re-roll already used")
	}

	taken := make(map[string]bool, len(s.users))
	for _, other := range s.users {
		if other.ID != id {
			taken[other.TeamID] = true
		}
	}

	var pool []TeamInfo
	for _, t := range availableTeams {
		if !taken[t.ID.String()] {
			pool = append(pool, t)
		}
	}
	if len(pool) == 0 {
		return nil, fmt.Errorf("no teams available for re-roll")
	}

	team := pool[rand.Intn(len(pool))]
	u.TeamID = team.ID.String()
	u.TeamName = team.NameEN
	u.TeamFA = team.NameFA
	u.FifaCode = team.FifaCode
	u.HasRerolled = true

	if err := s.save(); err != nil {
		return nil, fmt.Errorf("saving after re-roll: %w", err)
	}
	return u, nil
}

// save writes the current store to disk atomically.
func (s *UserStore) save() error {
	if err := os.MkdirAll("./data", 0755); err != nil {
		return err
	}
	list := make([]*UserProfile, 0, len(s.users))
	for _, u := range s.users {
		list = append(list, u)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	// Write to temp file then rename for atomicity.
	tmp := storeFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, storeFile)
}
