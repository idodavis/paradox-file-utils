// types.go is the wiki sidecar schema and Guide / patch RPC payloads.
package wiki

const (
	// FormatVersion is the on-disk schema of wiki sidecars. Mismatch → empty, no migration.
	FormatVersion = 1
	// License is the Paradox wiki editorial-text license shown in the UI.
	License   = "CC-BY-SA 3.0"
	htmlCap   = 1_500_000
	memberCap = 500
	infoBatch = 50
)

// Section is one MediaWiki TOC entry kept for heading extraction.
type Section struct {
	TocLevel int    `json:"tocLevel"`
	Line     string `json:"line"`
	Anchor   string `json:"anchor"`
}

// Page is one sanitized wiki page in a sidecar.
type Page struct {
	PageID      int       `json:"pageId"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Revid       int       `json:"revid"`
	FetchedAt   string    `json:"fetchedAt"`
	License     string    `json:"license"`
	Sections    []Section `json:"sections"`
	HTML        string    `json:"html"`
	ModdingHTML string    `json:"moddingHtml,omitempty"`
	Snippets    []string  `json:"snippets,omitempty"`
}

// header is the shared sidecar envelope.
type header struct {
	FormatVersion  int    `json:"formatVersion"`
	GameID         string `json:"gameId"`
	FetchedAt      string `json:"fetchedAt"`
	FetchedVersion string `json:"fetchedVersion"`
	FetchedMajor   int    `json:"fetchedMajor"`
	FetchedMinor   int    `json:"fetchedMinor"`
}

// Sidecar is wiki-docs-{gameId}.json or wiki-patches-{gameId}.json.
type Sidecar struct {
	header
	Pages []Page `json:"pages"`
}

// GuidePage is one tab in the Guide pane.
type GuidePage struct {
	Title    string    `json:"title"`
	HTML     string    `json:"html"`
	URL      string    `json:"url"`
	Snippets []string  `json:"snippets"`
	Sections []Section `json:"sections"`
}

// Guide is the WikiService.Guide payload. Rel/origin/kind come from Locate.
type Guide struct {
	Rel         string      `json:"rel"`
	Origin      string      `json:"origin"`
	OriginName  string      `json:"originName"`
	Kind        string      `json:"kind"`
	Pages       []GuidePage `json:"pages"`
	Attribution string      `json:"attribution"`
}

// Status is the WikiService.Status payload.
type Status struct {
	GameID         string `json:"gameId"`
	Present        bool   `json:"present"`
	FetchedAt      string `json:"fetchedAt"`
	FetchedVersion string `json:"fetchedVersion"`
	FetchedMajor   int    `json:"fetchedMajor"`
	FetchedMinor   int    `json:"fetchedMinor"`
	GuideCount     int    `json:"guideCount"`
	PatchCount     int    `json:"patchCount"`
}

// PatchEntry is one row in the Patch Notes version tree.
type PatchEntry struct {
	Title      string `json:"title"`
	Version    string `json:"version"`
	Parent     string `json:"parent"`
	URL        string `json:"url"`
	HasModding bool   `json:"hasModding"`
}

// PatchList is the WikiService.Patches payload.
type PatchList struct {
	GameID string       `json:"gameId"`
	Pages  []PatchEntry `json:"pages"`
}

// PatchPage is one patch notes body (All + optional Modding HTML).
type PatchPage struct {
	Title           string    `json:"title"`
	Version         string    `json:"version"`
	HTML            string    `json:"html"`
	ModdingHTML     string    `json:"moddingHtml"`
	URL             string    `json:"url"`
	HasModding      bool      `json:"hasModding"`
	Sections        []Section `json:"sections"`
	ModdingSections []Section `json:"moddingSections"`
}

// Progress reports wiki fetch counts for lang:scan-progress.
type Progress func(done, total int, kind string)
