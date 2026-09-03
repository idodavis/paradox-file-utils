// wikiservice.go is the Wails RPC shell for wiki guides, patch notes, and impact check.

package services

import (
	"path/filepath"
	"strings"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/session"
	"paradox-modding-tools/services/internal/wiki"
)

// WikiService exposes wiki sidecars over Wails. Scan still refreshes in-process.
type WikiService struct {
	Session *SessionService
}

// Guide returns Guide pane pages for the active file. Locate fail → empty, not an error.
func (s *WikiService) Guide(workspaceID, path string) (*wiki.Guide, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) *wiki.Guide {
		origin, rel, ok := sess.Locate(path)
		if !ok {
			return &wiki.Guide{}
		}
		slash := filepath.ToSlash(rel)
		kind := game.MatchExtract(sess.GameID, slash).Kind
		return &wiki.Guide{
			Rel:         slash,
			Origin:      origin,
			OriginName:  sess.OriginName(origin),
			Kind:        kind,
			Pages:       wiki.PagesFor(sess.GameID, kind, slash),
			Attribution: wiki.Attribution(sess.GameID),
		}
	})
}

// Status returns sidecar presence and counts for a game.
func (s *WikiService) Status(gameID string) (*wiki.Status, error) {
	st := wiki.StatusOf(gameID)
	return &st, nil
}

// Patches returns the Patch Notes version list, newest first.
func (s *WikiService) Patches(gameID string) (*wiki.PatchList, error) {
	list := wiki.Patches(gameID)
	return &list, nil
}

// PatchPage returns one patch notes body by title or version label.
func (s *WikiService) PatchPage(gameID, version string) (*wiki.PatchPage, error) {
	return wiki.PatchByVersion(gameID, version), nil
}

// ModAffected is the Impact Check heuristic against harvest + wiki Modding bullets.
func (s *WikiService) ModAffected(workspaceID, modID, from, to string) (*wiki.AffectedReport, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) *wiki.AffectedReport {
		return matchAffected(sess, modID, wiki.PatchRange(sess.GameID, from, to))
	})
}

func matchAffected(sess *session.Session, modID string, pages []wiki.Page) *wiki.AffectedReport {
	rep := &wiki.AffectedReport{}
	if modID == "" {
		return rep
	}
	name := sess.OriginName(modID)
	seen := map[string]bool{}
	for _, p := range pages {
		html := p.ModdingHTML
		if html == "" {
			rep.NotesWithoutMatch++
			continue
		}
		toks := wiki.Tokenize(html)
		hit := false
		for _, tk := range toks {
			row, ok := hitToken(sess, modID, name, p.Title, tk)
			if !ok {
				continue
			}
			key := row.Path + "\x00" + row.Token + "\x00" + row.PatchTitle
			if seen[key] {
				continue
			}
			seen[key] = true
			hit = true
			rep.Likely = append(rep.Likely, row)
		}
		if !hit {
			rep.NotesWithoutMatch++
		}
	}
	return rep
}

func hitToken(
	sess *session.Session, modID, originName, patchTitle string, tk wiki.Token,
) (wiki.AffectedRow, bool) {
	switch tk.Kind {
	case "path":
		if rel, path, ok := modPathPrefix(sess, modID, tk.Value); ok {
			return likely(rel, modID, originName, path, tk, patchTitle), true
		}
	case "rename":
		if rel, path, ok := tokenUsed(sess, modID, tk.Value); ok {
			return likely(rel, modID, originName, path, tk, patchTitle), true
		}
		if rel, path, ok := tokenUsed(sess, modID, tk.To); ok {
			return likely(rel, modID, originName, path, tk, patchTitle), true
		}
	case "ident":
		if rel, path, ok := tokenUsed(sess, modID, tk.Value); ok {
			return likely(rel, modID, originName, path, tk, patchTitle), true
		}
		if inEngineVocab(sess, tk.Value) {
			return wiki.AffectedRow{}, false
		}
	}
	return wiki.AffectedRow{}, false
}

func likely(rel, origin, name, path string, tk wiki.Token, patchTitle string) wiki.AffectedRow {
	return wiki.AffectedRow{
		Rel: rel, Origin: origin, OriginName: name, Path: path,
		Why: tk.Bullet, Token: tk.Value, PatchTitle: patchTitle,
	}
}

func tokenUsed(sess *session.Session, modID, token string) (rel, path string, ok bool) {
	if token == "" {
		return "", "", false
	}
	for _, d := range sess.ModDefsOf(token) {
		if d.Origin == modID {
			return sess.DisplayRel(d.Path), d.Path, true
		}
	}
	for _, r := range sess.RefsTo(token) {
		origin, rel, located := sess.Locate(r.Path)
		if located && origin == modID {
			return filepath.ToSlash(rel), r.Path, true
		}
	}
	return "", "", false
}

func modPathPrefix(sess *session.Session, modID, prefix string) (rel, path string, ok bool) {
	want := strings.ToLower(strings.ReplaceAll(prefix, "\\", "/"))
	for _, d := range sess.FindDefs("", 0, false, false) {
		if d.Origin != modID {
			continue
		}
		r := strings.ToLower(sess.DisplayRel(d.Path))
		if strings.Contains(r, want) || strings.HasPrefix(r, want) {
			return sess.DisplayRel(d.Path), d.Path, true
		}
	}
	return "", "", false
}

func inEngineVocab(sess *session.Session, tok string) bool {
	for _, kind := range []string{"effect", "trigger", "vocabulary"} {
		for _, v := range sess.Vocab(kind) {
			if v == tok {
				return true
			}
		}
	}
	return sess.TokenUsage(tok) != ""
}
