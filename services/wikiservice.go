// Package services provides backend services for the Paradox Modding Tools application.
package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/repos"

	"github.com/jmoiron/sqlx"
)

// WikiService fetches patch information from Paradox MediaWiki APIs.
type WikiService struct {
	DB     *sqlx.DB
	repo   *repos.WikiRepository
	client *http.Client
}

// PatchVersion is a lightweight version entry from the version index.
type PatchVersion struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

func (w *WikiService) getRepo() *repos.WikiRepository {
	if w.repo == nil {
		w.repo = repos.NewWikiRepository(w.DB)
	}
	return w.repo
}

func (w *WikiService) getClient() *http.Client {
	if w.client == nil {
		w.client = &http.Client{Timeout: 45 * time.Second}
	}
	return w.client
}

// setWikiHeaders applies browser-like headers to reduce Cloudflare false positives.
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

// ListVersions fetches the patch list from /Patches wiki page and returns versions.
func (w *WikiService) ListVersions(gameID string) ([]PatchVersion, error) {
	info := game.Get(gameID)
	if info == nil {
		return nil, fmt.Errorf("unknown game: %s", gameID)
	}

	apiURL := fmt.Sprintf("%s?action=parse&page=Patches&format=json&prop=text", info.WikiAPI)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	setWikiHeaders(req, info.WikiAPI)

	resp, err := w.getClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch patches page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if err := wikiHTTPError(resp.StatusCode, body); err != nil {
		return nil, err
	}

	text, err := parseWikiText(body)
	if err != nil {
		return nil, err
	}
	return w.parseVersionsFromHTML(text, gameID), nil
}

// parseVersionsFromHTML extracts patch versions from wiki HTML content.
func (w *WikiService) parseVersionsFromHTML(html, gameID string) []PatchVersion {
	info := game.Get(gameID)
	if info == nil {
		return nil
	}
	baseURL := strings.TrimSuffix(info.WikiAPI, "/api.php")

	re := regexp.MustCompile(`href="([^"]*Patch[_\s]*([\d.]+)[^"]*)"|>Patch\s*([\d.]+)<`)
	matches := re.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)
	var versions []PatchVersion

	for _, m := range matches {
		var version, href string
		if m[2] != "" {
			href = m[1]
			version = m[2]
		} else if m[3] != "" {
			version = m[3]
			href = fmt.Sprintf("/Patch_%s", version)
		}
		if version == "" || seen[version] {
			continue
		}
		seen[version] = true

		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = baseURL + href
		}
		versions = append(versions, PatchVersion{Version: version, URL: fullURL})
	}

	return versions
}

// GetPatchModdingSection fetches a patch page and extracts the Modding section HTML.
func (w *WikiService) GetPatchModdingSection(gameID, version string) (*repos.WikiPatch, error) {
	repo := w.getRepo()
	cached, err := repo.Get(gameID, version)
	if err == nil && cached != nil {
		return cached, nil
	}

	info := game.Get(gameID)
	if info == nil {
		return nil, fmt.Errorf("unknown game: %s", gameID)
	}

	pageTitle := fmt.Sprintf("Patch_%s", url.PathEscape(version))
	apiURL := fmt.Sprintf("%s?action=parse&page=%s&format=json&prop=text", info.WikiAPI, pageTitle)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	setWikiHeaders(req, info.WikiAPI)

	resp, err := w.getClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch patch page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if err := wikiHTTPError(resp.StatusCode, body); err != nil {
		return nil, err
	}

	text, err := parseWikiText(body)
	if err != nil {
		return nil, err
	}

	moddingHTML := w.extractModdingSection(text)
	baseURL := strings.TrimSuffix(info.WikiAPI, "/api.php")
	sourceURL := fmt.Sprintf("%s/Patch_%s", baseURL, version)

	patch := &repos.WikiPatch{
		GameID:      gameID,
		Version:     version,
		FetchedAt:   time.Now().UTC().Format(time.RFC3339),
		SourceURL:   sourceURL,
		HTMLContent: moddingHTML,
	}

	_ = repo.Upsert(patch)
	return patch, nil
}

// extractModdingSection extracts the Modding section from full wiki page HTML.
func (w *WikiService) extractModdingSection(html string) string {
	patterns := []string{
		`(?is)<span[^>]*id="Modding"[^>]*>.*?</span>.*?(<(?:ul|p|div)[^>]*>.*?)(?:<h[2-3]|<div class="printfooter"|$)`,
		`(?is)<h[2-3][^>]*>\s*Modding\s*</h[2-3]>.*?(<(?:ul|p|div)[^>]*>.*?)(?:<h[2-3]|<div class="printfooter"|$)`,
	}

	for _, p := range patterns {
		re := regexp.MustCompile(p)
		if m := re.FindStringSubmatch(html); len(m) > 1 {
			content := strings.TrimSpace(m[1])
			if content != "" {
				return "<div class=\"wiki-modding-section\">" + content + "</div>"
			}
		}
	}

	return ""
}

// ListCachedPatches returns all cached patches for a game.
func (w *WikiService) ListCachedPatches(gameID string) ([]repos.WikiPatch, error) {
	out, err := w.getRepo().ListByGame(gameID)
	if err != nil {
		return nil, fmt.Errorf("list cached patches: %w", err)
	}
	return out, nil
}

// wikiHTTPError maps non-OK wiki responses to clear user-facing errors.
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
		if snippet != "" && (strings.HasPrefix(snippet, "<") || strings.Contains(strings.ToLower(snippet), "<html")) {
			return fmt.Errorf("wiki API returned HTML instead of JSON (HTTP %d) — likely blocked or an error page", status)
		}
		return fmt.Errorf("wiki API returned %d", status)
	}
}

// parseWikiText unmarshals MediaWiki parse JSON and returns the HTML text blob.
func parseWikiText(body []byte) (string, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "", fmt.Errorf("wiki API returned an empty response")
	}
	if strings.HasPrefix(trimmed, "<") || strings.Contains(strings.ToLower(trimmed[:min(64, len(trimmed))]), "<html") {
		return "", fmt.Errorf("wiki API returned HTML instead of JSON (blocked or error page)")
	}
	var result struct {
		Parse struct {
			Text struct {
				Content string `json:"*"`
			} `json:"text"`
		} `json:"parse"`
		Error struct {
			Code string `json:"code"`
			Info string `json:"info"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("wiki response was not valid JSON: %w", err)
	}
	if result.Error.Code != "" {
		return "", fmt.Errorf("wiki error: %s - %s", result.Error.Code, result.Error.Info)
	}
	return result.Parse.Text.Content, nil
}
