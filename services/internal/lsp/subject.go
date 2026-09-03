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
	if d := resolveOfKindOwner(s, at.word, "script_param", at.paramOwner); d != nil {
		return subject{
			path: path, at: at, name: at.word, kind: "script_param",
			def: d, owner: at.paramOwner,
		}, true
	}
	if kind, name, ok := callKindDefAt(at.res, at.off); ok {
		if d := resolveOfKind(s, name, kind); d != nil {
			return subject{path: path, at: at, name: name, kind: kind, def: d}, true
		}
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

type kindInfo struct {
	label, hint string
}

var kindMeta = map[string]kindInfo{
	"loc_key":                 {"localization", "English text for this key"},
	"localization":            {"localization", "English text for this key"},
	"loc_value":               {"loc value", "Engine-supplied number, not a localization key"},
	"script_param":            {"script parameter", "Substituted at the call site"},
	"script_parameter":        {"script parameter", "Substituted at the call site"},
	"saved_scope":             {"saved scope", "Scope saved earlier in this chain"},
	"character_flag":          {"character flag", "Set on a character until an effect clears it"},
	"variable":                {"variable", "Last written value at this site"},
	"global_variable":         {"global variable", "Last written value at this site"},
	"local_variable":          {"local variable", "Last written value at this site"},
	"dead_character_variable": {"dead character variable", "Last written value at this site"},
	"script_value":            {"script value", "Named number referenced by CBs, decisions, and effects"},
	"game_rule":               {"game rule", "Game Rule to configure game settings"},
	"game_rule_setting":       {"game rule setting", "Option for a game rule"},
	"message":                 {"message", "Interface message"},
	"message_filter_types":    {"message filter", "Message filter type"},
	"message_filter":          {"message filter", "Message filter"},
	"scripted_trigger":        {"scripted trigger", "Macro for a trigger called elsewhere"},
	"scripted_effect":         {"scripted effect", "Macro for an effect called elsewhere"},
	"scripted_modifier":       {"scripted modifier", "Macro for a modifier called elsewhere"},
	"coat_of_arms":            {"coat of arms", "Coat Of Arms definition"},
	"data_function":           {"data function", "GUI data function "},
	"field":                   {"field", "Script field"},
	"on_action":               {"on action", "Fired when this game pulse runs"},
	"event":                   {"event", "Event definition"},
	"decision":                {"decision", "Decision definition"},
	"gui_type":                {"gui type", ""},
	"flag_definition":         {"flag definition", ""},
}

func kindLabel(t string) string {
	k := strings.ReplaceAll(game.CanonicalKind(t), " ", "_")
	if info, ok := kindMeta[k]; ok {
		return info.label
	}
	return strings.ReplaceAll(t, "_", " ")
}

func kindHint(kind string) string {
	k := strings.ReplaceAll(game.CanonicalKind(kind), " ", "_")
	if info, ok := kindMeta[k]; ok {
		return info.hint
	}
	if strings.HasSuffix(k, "_key") {
		return "Field on this " + strings.ReplaceAll(strings.TrimSuffix(k, "_key"), "_", " ")
	}
	return ""
}
