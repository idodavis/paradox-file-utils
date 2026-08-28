// Wiki doc cache: fetch Paradox Wiki API pages into user-data JSON (not git).
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"paradox-modding-tools/services/internal/game"
)

// WikiDocCache is one cached Paradox Wiki page with parsed identifier sections.
type WikiDocCache struct {
	GameID    string            `json:"gameId"`
	Page      string            `json:"page"`
	Revision  int               `json:"revision"`
	FetchedAt string            `json:"fetchedAt"`
	SourceURL string            `json:"sourceUrl"`
	Sections  map[string]string `json:"sections"`
}

var (
	wikiSectionRE = regexp.MustCompile(`(?m)^==+\s*(.+?)\s*==+\s*$`)
	wikiHTTP      = &http.Client{Timeout: 45 * time.Second}
)

// LoadCachedDocs merges cached wiki section maps for a game (disk only, no HTTP).
func LoadCachedDocs(gameID string) (map[string]string, error) {
	info := game.Get(gameID)
	if info == nil {
		return nil, fmt.Errorf("unknown game: %s", gameID)
	}
	out := map[string]string{}
	for _, page := range info.WikiDocPages {
		c, err := loadWikiDocCache(gameID, page)
		if err != nil || c == nil {
			continue
		}
		for k, v := range c.Sections {
			out[strings.ToLower(k)] = v
		}
	}
	return out, nil
}

// EnsureWikiDocs fetches wiki doc pages when revision changes; respects ctx cancel.
func EnsureWikiDocs(ctx context.Context, gameID string) error {
	info := game.Get(gameID)
	if info == nil {
		return fmt.Errorf("unknown game: %s", gameID)
	}
	for _, page := range info.WikiDocPages {
		if err := ctx.Err(); err != nil {
			return err
		}
		_ = ensureWikiPage(ctx, gameID, page, info.WikiAPI)
	}
	return nil
}

func wikiDocCachePath(gameID, page string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "Paradox Modding Tools", "wiki", gameID)
	return filepath.Join(dir, sanitizeWikiPage(page)+".json"), nil
}

func sanitizeWikiPage(page string) string {
	return strings.ReplaceAll(strings.TrimSpace(page), " ", "_")
}

func loadWikiDocCache(gameID, page string) (*WikiDocCache, error) {
	path, err := wikiDocCachePath(gameID, page)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c WikiDocCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func saveWikiDocCache(c *WikiDocCache) error {
	path, err := wikiDocCachePath(c.GameID, c.Page)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ensureWikiPage(ctx context.Context, gameID, page, wikiAPI string) error {
	rev, err := fetchWikiRevision(ctx, wikiAPI, page)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	}
	if old, err := loadWikiDocCache(gameID, page); err == nil && old != nil && old.Revision == rev {
		return nil
	}
	wikitext, sourceURL, err := fetchWikiWikitext(ctx, wikiAPI, page)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	}
	sections := parseWikiSections(wikitext)
	c := &WikiDocCache{
		GameID:    gameID,
		Page:      page,
		Revision:  rev,
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		SourceURL: sourceURL,
		Sections:  sections,
	}
	return saveWikiDocCache(c)
}

func setWikiHeaders(req *http.Request, wikiAPI string) {
	base := strings.TrimSuffix(wikiAPI, "/api.php")
	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "+
			"(KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 ParadoxModdingTools/1.0")
	req.Header.Set("Accept", "application/json,text/javascript,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if base != "" {
		req.Header.Set("Referer", base+"/")
		req.Header.Set("Origin", base)
	}
}

func wikiHTTPError(status int, body []byte) error {
	if status == http.StatusOK {
		return nil
	}
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 120 {
		snippet = snippet[:120] + "…"
	}
	switch status {
	case http.StatusForbidden, http.StatusUnavailableForLegalReasons:
		return fmt.Errorf(
			"wiki API blocked (HTTP %d, likely Cloudflare). "+
				"Open the wiki in a browser, or use cached patches if available",
			status,
		)
	case http.StatusTooManyRequests:
		return fmt.Errorf("wiki API rate-limited the request (429); try again later")
	default:
		if snippet != "" && (strings.HasPrefix(snippet, "<") ||
			strings.Contains(strings.ToLower(snippet), "<html")) {
			return fmt.Errorf(
				"wiki API returned HTML instead of JSON (HTTP %d) — likely blocked or an error page",
				status,
			)
		}
		return fmt.Errorf("wiki API returned %d", status)
	}
}

func fetchWikiRevision(ctx context.Context, wikiAPI, page string) (int, error) {
	q := url.Values{
		"action": {"query"},
		"format": {"json"},
		"prop":   {"revisions"},
		"rvprop": {"ids"},
		"titles": {page},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wikiAPI+"?"+q.Encode(), nil)
	if err != nil {
		return 0, err
	}
	setWikiHeaders(req, wikiAPI)
	resp, err := wikiHTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if err := wikiHTTPError(resp.StatusCode, body); err != nil {
		return 0, err
	}
	var parsed struct {
		Query struct {
			Pages map[string]struct {
				Revisions []struct {
					RevID int `json:"revid"`
				} `json:"revisions"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	for _, p := range parsed.Query.Pages {
		if len(p.Revisions) > 0 {
			return p.Revisions[0].RevID, nil
		}
	}
	return 0, fmt.Errorf("no revision for %s", page)
}

func fetchWikiWikitext(ctx context.Context, wikiAPI, page string) (string, string, error) {
	q := url.Values{
		"action": {"parse"},
		"format": {"json"},
		"page":   {page},
		"prop":   {"wikitext"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wikiAPI+"?"+q.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	setWikiHeaders(req, wikiAPI)
	resp, err := wikiHTTP.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if err := wikiHTTPError(resp.StatusCode, body); err != nil {
		return "", "", err
	}
	var parsed struct {
		Parse struct {
			Title    string `json:"title"`
			Wikitext struct {
				Star string `json:"*"`
			} `json:"wikitext"`
		} `json:"parse"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", err
	}
	base := strings.TrimSuffix(wikiAPI, "/api.php")
	sourceURL := base + "/" + url.PathEscape(strings.ReplaceAll(page, " ", "_"))
	return parsed.Parse.Wikitext.Star, sourceURL, nil
}

// parseWikiSections splits wikitext on == headings; section title → body text.
func parseWikiSections(wikitext string) map[string]string {
	out := map[string]string{}
	parts := wikiSectionRE.Split(wikitext, -1)
	titles := wikiSectionRE.FindAllStringSubmatch(wikitext, -1)
	for i, title := range titles {
		if len(title) < 2 || i+1 >= len(parts) {
			continue
		}
		name := strings.Fields(title[1])[0]
		body := strings.TrimSpace(cleanWikiBody(parts[i+1]))
		if name != "" && body != "" {
			out[strings.ToLower(name)] = body
		}
	}
	return out
}

func cleanWikiBody(s string) string {
	s = regexp.MustCompile(`'''+?`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`<ref[^>]*>.*?</ref>`).ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
