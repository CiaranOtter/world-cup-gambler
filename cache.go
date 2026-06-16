// cache.go
package main

import (
	"log"
	"sync"
	"time"
)

// DataCache holds the latest tournament data fetched from
// worldcup26.ir, refreshed on a timer. Reads are served from memory
// so handlers never block on an upstream request.
type DataCache struct {
	mu sync.RWMutex

	matches  []Match
	teams    map[string]TeamInfo // keyed by team ID
	stadiums map[string]Stadium  // keyed by stadium ID
	groups   []GroupTable

	lastUpdated time.Time
	lastError   error

	client *WorldCupClient
}

func NewDataCache(client *WorldCupClient) *DataCache {
	return &DataCache{
		client:   client,
		teams:    make(map[string]TeamInfo),
		stadiums: make(map[string]Stadium),
	}
}

// Start performs an initial fetch and then refreshes on the given
// interval in the background. Live matches change frequently during
// the tournament, so a short interval (e.g. 30s) is reasonable.
func (c *DataCache) Start(interval time.Duration) {
	c.refresh()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			c.refresh()
		}
	}()
}

func (c *DataCache) refresh() {
	teams, err := c.client.GetTeams()
	if err != nil {
		log.Printf("cache: fetching teams: %v", err)
		c.setError(err)
		return
	}

	stadiums, err := c.client.GetStadiums()
	if err != nil {
		log.Printf("cache: fetching stadiums: %v", err)
		c.setError(err)
		return
	}

	groups, err := c.client.GetGroups()
	if err != nil {
		log.Printf("cache: fetching groups: %v", err)
		// Non-fatal: groups are supplementary. Continue with what we have.
	}

	rawMatches, err := c.client.GetMatches()
	if err != nil {
		log.Printf("cache: fetching matches: %v", err)
		c.setError(err)
		return
	}

	teamsByID := make(map[string]TeamInfo, len(teams))
	for _, t := range teams {
		teamsByID[t.ID.String()] = t
	}

	stadiumsByID := make(map[string]Stadium, len(stadiums))
	for _, s := range stadiums {
		stadiumsByID[s.ID.String()] = s
	}

	matches := make([]Match, 0, len(rawMatches))
	for _, rm := range rawMatches {
		m := rm.ToMatch(teamsByID)
		if stadium, ok := stadiumsByID[m.StadiumID]; ok {
			m.Stadium = stadium
		}
		matches = append(matches, m)
	}

	c.mu.Lock()
	c.matches = matches
	c.teams = teamsByID
	c.stadiums = stadiumsByID
	if groups != nil {
		c.groups = groups
	}
	c.lastUpdated = time.Now()
	c.lastError = nil
	c.mu.Unlock()

	log.Printf("cache: refreshed (%d matches, %d teams, %d stadiums, %d groups)",
		len(matches), len(teamsByID), len(stadiumsByID), len(c.groups))
}

func (c *DataCache) setError(err error) {
	c.mu.Lock()
	c.lastError = err
	c.mu.Unlock()
}

// Matches returns a copy of the cached match list.
func (c *DataCache) Matches() []Match {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Match, len(c.matches))
	copy(out, c.matches)
	return out
}

// MatchByID returns a single match by ID, if present.
func (c *DataCache) MatchByID(id string) (Match, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, m := range c.matches {
		if m.ID == id {
			return m, true
		}
	}
	return Match{}, false
}

// Teams returns a copy of the cached teams, keyed by ID.
func (c *DataCache) Teams() map[string]TeamInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]TeamInfo, len(c.teams))
	for k, v := range c.teams {
		out[k] = v
	}
	return out
}

// Stadiums returns a copy of the cached stadiums, keyed by ID.
func (c *DataCache) Stadiums() map[string]Stadium {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]Stadium, len(c.stadiums))
	for k, v := range c.stadiums {
		out[k] = v
	}
	return out
}

// Groups returns a copy of the cached group standings.
func (c *DataCache) Groups() []GroupTable {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]GroupTable, len(c.groups))
	copy(out, c.groups)
	return out
}

// Status returns the last successful update time and any error from
// the most recent refresh attempt.
func (c *DataCache) Status() (time.Time, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUpdated, c.lastError
}
