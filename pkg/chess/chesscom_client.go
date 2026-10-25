package chess

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kbelludoo/mbvm-db/pkg/mbvmdb"
)

// ChessComClient interacts with the official Chess.com PubAPI.
type ChessComClient struct {
	httpClient *http.Client
	userAgent  string
}

// NewChessComClient creates a client configured with respectful headers for Chess.com.
func NewChessComClient() *ChessComClient {
	return &ChessComClient{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		userAgent: "chess-mbvmdb/1.0 (author: kbelludoo; embedded micro-bytecode graph engine)",
	}
}

// PlayerProfile represents basic public data for a Chess.com player.
type PlayerProfile struct {
	Username string `json:"username"`
	PlayerID int64  `json:"player_id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	URL      string `json:"url"`
	Name     string `json:"name"`
	Country  string `json:"country"`
}

// MonthlyArchive represents the list of games returned for a given month.
type MonthlyArchive struct {
	Games []struct {
		URL       string `json:"url"`
		PGN       string `json:"pgn"`
		TimeControl string `json:"time_control"`
		TimeClass string `json:"time_class"`
		Rules     string `json:"rules"`
		White     struct {
			Rating   int    `json:"rating"`
			Result   string `json:"result"`
			Username string `json:"username"`
		} `json:"white"`
		Black struct {
			Rating   int    `json:"rating"`
			Result   string `json:"result"`
			Username string `json:"username"`
		} `json:"black"`
		ECO string `json:"eco"`
	} `json:"games"`
}

// GetPlayerProfile fetches the player profile from Chess.com.
func (c *ChessComClient) GetPlayerProfile(username string) (*PlayerProfile, error) {
	url := fmt.Sprintf("https://api.chess.com/pub/player/%s", strings.ToLower(username))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chess.com api request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chess.com api returned status %d for player %s", resp.StatusCode, username)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var profile PlayerProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile json: %w", err)
	}

	return &profile, nil
}

// GetPlayerArchives fetches the available monthly archive URLs for a player.
func (c *ChessComClient) GetPlayerArchives(username string) ([]string, error) {
	url := fmt.Sprintf("https://api.chess.com/pub/player/%s/games/archives", strings.ToLower(username))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chess.com api archives returned status %d for %s", resp.StatusCode, username)
	}

	var data struct {
		Archives []string `json:"archives"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data.Archives, nil
}

// FetchRecentGames downloads games from the latest archive for a player and converts to mbvmdb items.
func (c *ChessComClient) FetchRecentGames(username string, maxGames int) ([]mbvmdb.ChessGameItem, error) {
	archives, err := c.GetPlayerArchives(username)
	if err != nil {
		return nil, err
	}
	if len(archives) == 0 {
		return nil, fmt.Errorf("no game archives found for %s", username)
	}

	// Use latest month archive
	latestArchiveURL := archives[len(archives)-1]
	req, err := http.NewRequest("GET", latestArchiveURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chess.com games archive returned status %d", resp.StatusCode)
	}

	var archive MonthlyArchive
	if err := json.NewDecoder(resp.Body).Decode(&archive); err != nil {
		return nil, err
	}

	var result []mbvmdb.ChessGameItem
	count := 0
	for i := len(archive.Games) - 1; i >= 0; i-- {
		g := archive.Games[i]
		if g.Rules != "chess" {
			continue // skip chess960 or variants for now
		}

		parsedGames := mbvmdb.ParsePGN(g.PGN)
		if len(parsedGames) > 0 {
			pg := parsedGames[0]
			pg.URL = g.URL
			pg.White = g.White.Username
			pg.WhiteRating = g.White.Rating
			pg.Black = g.Black.Username
			pg.BlackRating = g.Black.Rating
			pg.TimeClass = g.TimeClass
			pg.PGN = g.PGN
			if g.ECO != "" {
				pg.ECO = g.ECO
			}
			result = append(result, pg)
			count++
			if maxGames > 0 && count >= maxGames {
				break
			}
		}
	}

	return result, nil
}
