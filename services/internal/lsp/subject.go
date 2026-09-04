// subject.go resolves the named thing under the cursor for hover, F12, and refs.
package lsp

import (
	"strings"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/parser/loc"
	"paradox-modding-tools/services/internal/session"
)

// subject is the identifier at a cursor: a def, ephemeral name, field, or loc value.
type subject struct {
	path      string
	at        atPos
	name      string
	kind      string
	def       *catalog.Def
	owner     string
	local     bool
	fieldKey  bool
	locFilter string
	spanStart int
	spanEnd   int
}

func callKindSub(s *session.Session, path string, at atPos) (subject, bool) {
	kind, name, a, ok := callKindDefAt(at.res, at.off)
	if !ok || name == "" {
		return subject{}, false
	}
	d := resolveOfKind(s, name, kind)
	if d != nil {
		for _, cand := range s.ModDefsOf(name) {
			if game.CanonicalKind(cand.Kind) == game.CanonicalKind(kind) &&
				session.SamePath(cand.Path, path) {
				c := cand
				d = &c
				break
			}
		}
	}
	if d == nil && a != nil {
		origin, _, _ := s.Locate(path)
		d = &catalog.Def{
			Kind: kind, Key: name, Path: path, Origin: origin,
			Start: a.Key.Range.Start, End: a.Key.Range.End,
			Line: at.res.Lines().PositionAt(a.Key.Range.Start).Line,
		}
	}
	return subject{path: path, at: at, name: name, kind: kind, def: d}, true
}

func ephemeralSub(s *session.Session, path string, at atPos, name, kind string) (subject, bool) {
	if name == "" || kind == "" {
		return subject{}, false
	}
	return subject{
		path: path, at: at, name: name, kind: kind,
		def: resolveOfKind(s, name, kind),
	}, true
}

// subjectAt reports the named thing at (line, UTF-8 column), or false if none.
func subjectAt(s *session.Session, path string, line, col int) (subject, bool) {
	if s.KindFor(path) == "loc" {
		return locSubject(s, path, line, col)
	}
	at, ok := resolveAt(s, path, line, col)
	if !ok {
		return subject{}, false
	}
	if name, ok := scopeRefAt(at.src, at.off); ok {
		return ephemeralSub(s, path, at, name, "saved_scope")
	}
	if at.saveOK {
		return ephemeralSub(s, path, at, at.saveName, "saved_scope")
	}
	if name, pStart, pEnd, ok := game.ScriptParamSpan(at.src, at.off); ok {
		d := resolveOfKindOwner(s, name, "script_param", at.paramOwner)
		if d == nil {
			origin, _, _ := s.Locate(path)
			d = &catalog.Def{
				Kind: "script_param", Key: name, Path: path, Origin: origin,
				Start: pStart, End: pEnd, OwnerKey: at.paramOwner,
				Line: at.res.Lines().PositionAt(pStart).Line,
			}
		}
		return subject{
			path: path, at: at, name: name, kind: "script_param",
			def: d, owner: at.paramOwner,
		}, true
	}
	if d := resolveOfKindOwner(s, at.word, "script_param", at.paramOwner); d != nil {
		return subject{
			path: path, at: at, name: at.word, kind: "script_param",
			def: d, owner: at.paramOwner,
		}, true
	}
	if sub, ok := callKindSub(s, path, at); ok {
		return sub, true
	}
	if a := at.assign; a != nil {
		if p, ok := game.ParsePrefixed(a.Key.Text); ok && game.IsSavedScopePrefix(p) {
			return ephemeralSub(s, path, at, p.Name, "saved_scope")
		}
		if isLocalDefFile(s, path, at.kind) {
			return subject{
				path: path, at: at, name: a.Key.Text, kind: at.kind, local: true,
			}, true
		}
		if d := resolveNonLoc(s, a.Key.Text); d != nil {
			return subject{path: path, at: at, name: d.Key, kind: d.Kind, def: d}, true
		}
		if ck := game.CanonicalKind(a.Key.Text); game.IsCallKind(ck) {
			return subject{path: path, at: at, name: a.Key.Text, kind: ck}, true
		}
		if ck := memberSetsFor(s, "").catalogKind(a.Key.Text); ck != "" {
			return subject{path: path, at: at, name: a.Key.Text, kind: ck}, true
		}
		docs := s.FieldDoc(a.Key.Text, at.kind)
		m := memberSetsFor(s, at.kind)
		if docs != "" || m.structKeys[a.Key.Text] ||
			m.structKeys[strings.ToLower(a.Key.Text)] {
			return subject{
				path: path, at: at, name: a.Key.Text, kind: at.kind, fieldKey: true,
			}, true
		}
	}
	if at.word == "" {
		return subject{}, false
	}
	if p, ok := game.ParsePrefixed(at.word); ok {
		if kind := game.PrefixKind(p.Prefix); kind != "" {
			return ephemeralSub(s, path, at, p.Name, kind)
		}
	}
	if at.slotKey != "" {
		if k := game.RefFieldKind(s.GameID, at.slotKey); k != "" {
			if game.IsEphemeral(k) {
				return ephemeralSub(s, path, at, at.word, k)
			}
			if loc.Classify(at.slotKey) == loc.PropNone {
				if d := resolveOfKind(s, at.word, k); d != nil {
					return subject{
						path: path, at: at, name: at.word, kind: k, def: d,
					}, true
				}
			}
		}
		if at.msgType {
			if d := resolveOfKind(s, at.word, "message"); d != nil {
				return subject{
					path: path, at: at, name: at.word, kind: "message", def: d,
				}, true
			}
		}
	}
	if d := s.Resolve(at.word); d != nil {
		return subject{path: path, at: at, name: d.Key, kind: d.Kind, def: d}, true
	}
	if locDefined(s, at.word) {
		if file, ln, origin, ok := s.LocSite(at.word); ok {
			d := &catalog.Def{
				Kind: "loc_key", Key: at.word, Path: file, Line: ln, Origin: origin,
			}
			return subject{
				path: path, at: at, name: at.word, kind: "loc_key", def: d,
			}, true
		}
	}
	if ck := memberSetsFor(s, "").catalogKind(at.word); ck != "" {
		return subject{path: path, at: at, name: at.word, kind: ck}, true
	}
	for _, k := range s.Vocab("datafunction") {
		if k == at.word {
			return subject{
				path: path, at: at, name: at.word, kind: "data_function",
			}, true
		}
	}
	if ck := game.CanonicalKind(at.word); game.IsCallKind(ck) {
		return subject{path: path, at: at, name: at.word, kind: ck}, true
	}
	if extra := s.FieldDoc(at.word, ""); extra != "" {
		return subject{path: path, at: at, name: at.word, kind: "field"}, true
	}
	if at.word != "" {
		return subject{path: path, at: at, name: at.word}, true
	}
	return subject{}, false
}

func locSubject(s *session.Session, path string, line, col int) (subject, bool) {
	src := s.FileText(path)
	if src == "" {
		return subject{}, false
	}
	off := jomini.NewLineIndex(src).OffsetAt(line, col)
	if ip, ok := locInterpAt(src, off); ok && game.IsLocEngineValue(ip.Key, ip.Filter) {
		return subject{
			path: path, name: ip.Key, kind: "loc_value", locFilter: ip.Filter,
		}, true
	}
	key, start, end := locKeySpan(src, off)
	if key == "" {
		return subject{}, false
	}
	sub := subject{
		path: path, name: key, kind: "loc_key", spanStart: start, spanEnd: end,
	}
	if file, ln, origin, ok := s.LocSite(key); ok {
		sub.def = &catalog.Def{
			Kind: "loc_key", Key: key, Path: file, Line: ln, Origin: origin,
		}
	}
	return sub, true
}

func scopeRefAt(src string, off int) (name string, ok bool) {
	word, wStart, wEnd := jomini.Result{Src: src}.TokenAt(off)
	if word == "" {
		return "", false
	}
	if wEnd < len(src) && src[wEnd] == ':' && word == game.ScopePrefix {
		n, _, _ := jomini.Result{Src: src}.TokenAt(wEnd + 1)
		if n == "" {
			return "", false
		}
		return n, true
	}
	if wStart > 0 && src[wStart-1] == ':' {
		pre, _, _ := jomini.Result{Src: src}.TokenAt(wStart - 2)
		if pre == game.ScopePrefix {
			return word, true
		}
	}
	p, pok := game.ParsePrefixed(word)
	if pok && game.IsSavedScopePrefix(p) {
		return p.Name, true
	}
	return "", false
}
