// lookup.go scores persisted wiki pages for Guide and Patch Center queries.
package wiki

import (
	"fmt"
	"slices"
	"strings"
)

// rankedPage is one sidecar page with its lookup score and language-tab flag.
type rankedPage struct {
	page  Page
	score int
	lang  bool
}

func pagesFor(gameID, kind, rel string) []GuidePage {
	f, err := LoadGuides(gameID)
	if err != nil || f == nil {
		return nil
	}
	m := fileTokens(kind, rel)
	script := isScriptKind(kind)
	cands := make([]rankedPage, 0, len(f.Pages))
	for _, p := range f.Pages {
		sc := scoreTitle(p.Title, m)
		lang := script && isLangStem(stemTitle(p.Title))
		if sc == 0 && !lang {
			continue
		}
		cands = append(cands, rankedPage{page: p, score: sc, lang: lang})
	}
	if len(cands) == 0 {
		return nil
	}
	prim := -1
	for i, c := range cands {
		if c.score <= 0 {
			continue
		}
		if prim < 0 || betterPage(c.page.Title, c.score, cands[prim].page.Title, cands[prim].score) {
			prim = i
		}
	}
	var primary Page
	hasPrimary := prim >= 0
	if hasPrimary {
		primary = cands[prim].page
	}
	slices.SortFunc(cands, func(a, b rankedPage) int {
		if a.lang != b.lang {
			if a.lang {
				return -1
			}
			return 1
		}
		if a.score != b.score {
			return b.score - a.score
		}
		return strings.Compare(a.page.Title, b.page.Title)
	})
	seen := map[string]bool{}
	var out []GuidePage
	add := func(p Page) {
		k := normTitle(p.Title)
		if seen[k] || len(out) >= guideTabCap {
			return
		}
		seen[k] = true
		html, secs := reshape(p.HTML)
		out = append(out, GuidePage{
			Title:    p.Title,
			HTML:     html,
			URL:      p.URL,
			Snippets: slices.Clone(p.Snippets),
			Sections: secs,
		})
	}
	if hasPrimary {
		add(primary)
	}
	for _, c := range cands {
		add(c.page)
	}
	return out
}

// PagesFor returns Guide tabs for an extract kind + relative path.
// Missing sidecar or unmatched file → empty pages, no error.
func PagesFor(gameID, kind, rel string) []GuidePage {
	return pagesFor(gameID, kind, rel)
}

// StatusOf returns sidecar presence and counts.
func StatusOf(gameID string) Status {
	st := Status{GameID: gameID, Present: HasSidecar(gameID)}
	if d, err := LoadGuides(gameID); err == nil && d != nil {
		st.FetchedAt = d.FetchedAt
		st.FetchedVersion = d.FetchedVersion
		st.FetchedMajor = d.FetchedMajor
		st.FetchedMinor = d.FetchedMinor
		st.GuideCount = len(d.Pages)
	}
	if p, err := LoadPatches(gameID); err == nil && p != nil {
		st.PatchCount = len(p.Pages)
		if st.FetchedAt == "" {
			st.FetchedAt = p.FetchedAt
			st.FetchedVersion = p.FetchedVersion
			st.FetchedMajor = p.FetchedMajor
			st.FetchedMinor = p.FetchedMinor
		}
	}
	return st
}

func patchEntry(p Page) PatchEntry {
	e := PatchEntry{Title: p.Title, URL: p.URL, HasModding: p.ModdingHTML != ""}
	if pv, ok := parsePatchTitle(p.Title); ok {
		e.Version = pv.Label
		if pv.Patch > 0 || pv.X {
			e.Parent = itoa2(pv.Major, pv.Minor)
		}
	}
	return e
}

func itoa2(maj, min int) string {
	return fmt.Sprintf("%d.%d", maj, min)
}

// Patches returns patch notes entries, newest first.
func Patches(gameID string) PatchList {
	out := PatchList{GameID: gameID}
	f, err := LoadPatches(gameID)
	if err != nil || f == nil {
		return out
	}
	type row struct {
		e  PatchEntry
		pv patchVer
		ok bool
	}
	rows := make([]row, 0, len(f.Pages))
	for _, p := range f.Pages {
		pv, ok := parsePatchTitle(p.Title)
		rows = append(rows, row{e: patchEntry(p), pv: pv, ok: ok})
	}
	slices.SortFunc(rows, func(a, b row) int {
		if a.ok && b.ok {
			return -cmpPatch(a.pv, b.pv)
		}
		if a.ok != b.ok {
			if a.ok {
				return -1
			}
			return 1
		}
		return strings.Compare(b.e.Title, a.e.Title)
	})
	for _, r := range rows {
		out.Pages = append(out.Pages, r.e)
	}
	return out
}

// PatchByVersion finds a patch page by title or version label (e.g. "1.16").
func PatchByVersion(gameID, want string) *PatchPage {
	f, err := LoadPatches(gameID)
	if err != nil || f == nil {
		return nil
	}
	wantN := normTitle(want)
	wantN = strings.TrimPrefix(wantN, "patch ")
	var fallback *Page
	for i := range f.Pages {
		p := &f.Pages[i]
		n := normTitle(p.Title)
		if n == wantN || n == "patch "+wantN {
			return patchPageOf(p)
		}
		if pv, ok := parsePatchTitle(p.Title); ok && pv.Label == want {
			return patchPageOf(p)
		}
		if fallback == nil && strings.Contains(n, wantN) {
			fallback = p
		}
	}
	if fallback != nil {
		return patchPageOf(fallback)
	}
	return nil
}

func patchPageOf(p *Page) *PatchPage {
	pv, _ := parsePatchTitle(p.Title)
	html, secs := reshape(p.HTML)
	mod := extractModding(html)
	_, modSecs := reshape(mod)
	return &PatchPage{
		Title:           p.Title,
		Version:         pv.Label,
		HTML:            html,
		ModdingHTML:     mod,
		URL:             p.URL,
		HasModding:      mod != "",
		Sections:        secs,
		ModdingSections: modSecs,
	}
}
