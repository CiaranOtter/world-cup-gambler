// worldcup_client.go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const worldCupAPIBase = "https://worldcup26.ir"

// WorldCupClient talks to the worldcup26.ir API.
// Authorization is optional: GET endpoints work without a token in demo
// mode. If WORLDCUP_API_TOKEN is set, it's sent as a Bearer token.
type WorldCupClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewWorldCupClient() *WorldCupClient {
	return &WorldCupClient{
		baseURL: worldCupAPIBase,
		token:   os.Getenv("WORLDCUP_API_TOKEN"), // optional
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// getRaw performs a GET request and returns the raw response body.
func (c *WorldCupClient) getRaw(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s response: %w", path, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("worldcup26.ir %s returned status %d: %s", path, resp.StatusCode, truncate(body, 200))
	}

	return body, nil
}

// getList fetches path and decodes it into v, which must be a pointer
// to a slice. The worldcup26.ir API sometimes returns a bare JSON array
// and sometimes wraps it in an object (e.g. {"data": [...]},
// {"teams": [...]}, {"games": [...]}). This handles both shapes by
// trying a direct array decode first, then falling back to scanning a
// wrapper object for the first array-valued field.
func (c *WorldCupClient) getList(path string, v interface{}) error {
	body, err := c.getRaw(path)
	if err != nil {
		return err
	}

	// Try decoding as a bare array first.
	if err := json.Unmarshal(body, v); err == nil {
		return nil
	}

	// Fall back: treat as an object and look for an array field.
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return fmt.Errorf("decoding %s response (not an array or object): %w; body: %s", path, err, truncate(body, 200))
	}

	// Common wrapper keys, checked in order of likelihood.
	candidateKeys := []string{"data", "teams", "games", "matches", "stadiums", "groups", "results", "items"}
	for _, key := range candidateKeys {
		raw, ok := wrapper[key]
		if !ok {
			continue
		}
		if err := json.Unmarshal(raw, v); err == nil {
			return nil
		}
	}

	// Last resort: scan every field for the first one that decodes
	// successfully as the target slice type.
	for _, raw := range wrapper {
		if err := json.Unmarshal(raw, v); err == nil {
			return nil
		}
	}

	return fmt.Errorf("decoding %s response: no array field found in object; body: %s", path, truncate(body, 200))
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}

// GetMatches fetches all 104 matches from /get/games.
func (c *WorldCupClient) GetMatches() ([]rawMatch, error) {
	var raw []rawMatch
	if err := c.getList("/get/games", &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// GetTeams fetches all 48 teams from /get/teams.
func (c *WorldCupClient) GetTeams() ([]TeamInfo, error) {
	var teams []TeamInfo
	if err := c.getList("/get/teams", &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

// GetStadiums fetches all 16 stadiums from /get/stadiums.
func (c *WorldCupClient) GetStadiums() ([]Stadium, error) {
	var stadiums []Stadium
	if err := c.getList("/get/stadiums", &stadiums); err != nil {
		return nil, err
	}
	return stadiums, nil
}

// GetGroups fetches all 12 group standings tables from /get/groups.
func (c *WorldCupClient) GetGroups() ([]GroupTable, error) {
	var groups []GroupTable
	if err := c.getList("/get/groups", &groups); err != nil {
		return nil, err
	}
	return groups, nil
}
