// types.go holds graph DTOs (no coordinates) and the shared parse/block/loc
// helpers every view uses against a live session.

package graph

import (
	"regexp"
	"strings"

	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

// EventGraphParams selects which definitions to draw.
type EventGraphParams struct {
	Root      string `json:"root,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	ModRoot   string `json:"modRoot,omitempty"`
	MaxNodes  int    `json:"maxNodes,omitempty"`
	Themes    bool   `json:"themes,omitempty"`
}

// EventGraph is a coordinate-free node/edge payload for frontend layout.
type EventGraph struct {
	Nodes       []EventGraphNode      `json:"nodes"`
	Edges       []EventGraphEdge      `json:"edges"`
	Truncated   bool                  `json:"truncated"`
	Suggestions EventGraphSuggestions `json:"suggestions"`
	EmptyReason string                `json:"emptyReason,omitempty"`
}

// EventGraphSuggestions is the mod-side catalog for the query box.
type EventGraphSuggestions struct {
	IDs        []string `json:"ids"`
	Namespaces []string `json:"namespaces"`
}

// EventGraphStep is one card row in execution order.
type EventGraphStep struct {
	Phase string `json:"phase"`
	Index *int   `json:"index,omitempty"`
	Text  string `json:"text,omitempty"`
	Line  int    `json:"line"`
}

// EventGraphNode is one graph card. There is no x/y: layout is frontend/dagre.
type EventGraphNode struct {
	ID             string           `json:"id"`
	Kind           string           `json:"kind"`
	Source         string           `json:"source"`
	File           string           `json:"file,omitempty"`
	Line           int              `json:"line,omitempty"`
	Title          string           `json:"title,omitempty"`
	Theme          string           `json:"theme,omitempty"`
	Options        int              `json:"options,omitempty"`
	TriggerSummary string           `json:"triggerSummary,omitempty"`
	Fires          int              `json:"fires,omitempty"`
	Steps          []EventGraphStep `json:"steps,omitempty"`
}

// EventGraphEdge is a directed link. Via names the referencing field or hop chain.
type EventGraphEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Via      string `json:"via"`
	Label    string `json:"label,omitempty"`
	Phase    string `json:"phase,omitempty"`
	FromLine *int   `json:"fromLine,omitempty"`
	Delay    string `json:"delay,omitempty"`
	Weight   *int   `json:"weight,omitempty"`
}

// EventLocField is a resolved localization key.
type EventLocField struct {
	Key     string `json:"key"`
	Text    string `json:"text,omitempty"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Dynamic bool   `json:"dynamic,omitempty"`
}

// EventScriptLine is one rendered pseudo-script line.
type EventScriptLine struct {
	Depth int    `json:"depth"`
	Text  string `json:"text"`
	Line  int    `json:"line"`
}

// EventStepTarget is an event/on_action a block hands control to.
type EventStepTarget struct {
	Via        string            `json:"via"`
	Name       string            `json:"name"`
	Kind       string            `json:"kind"`
	Line       int               `json:"line"`
	File       string            `json:"file,omitempty"`
	DefLine    int               `json:"defLine,omitempty"`
	DefCount   int               `json:"defCount,omitempty"`
	Fires      []EventStepTarget `json:"fires,omitempty"`
	FiresTotal int               `json:"firesTotal,omitempty"`
}

// EventFieldInfo is a top-level scalar in an event or option body.
type EventFieldInfo struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Line   int    `json:"line"`
	Quoted bool   `json:"quoted,omitempty"`
}

// EventSectionInfo is one named event block (trigger, immediate, …).
type EventSectionInfo struct {
	Name         string            `json:"name"`
	Line         int               `json:"line"`
	Keys         []string          `json:"keys"`
	Lines        []EventScriptLine `json:"lines"`
	TotalLines   int               `json:"totalLines"`
	Targets      []EventStepTarget `json:"targets"`
	TargetsTotal int               `json:"targetsTotal"`
}

// EventGateInfo is a rendered trigger or ai_chance block.
type EventGateInfo struct {
	Line       int               `json:"line"`
	Lines      []EventScriptLine `json:"lines"`
	TotalLines int               `json:"totalLines"`
}

// EventOptionInfo is one option = { } block.
type EventOptionInfo struct {
	Line         int               `json:"line"`
	Fields       []EventFieldInfo  `json:"fields"`
	Name         *EventLocField    `json:"name,omitempty"`
	EffectKeys   []string          `json:"effectKeys"`
	HasTrigger   bool              `json:"hasTrigger"`
	HasAiChance  bool              `json:"hasAiChance"`
	Trigger      *EventGateInfo    `json:"trigger,omitempty"`
	AiChance     *EventGateInfo    `json:"aiChance,omitempty"`
	Lines        []EventScriptLine `json:"lines"`
	TotalLines   int               `json:"totalLines"`
	Targets      []EventStepTarget `json:"targets"`
	TargetsTotal int               `json:"targetsTotal"`
}

// EventRefInfo is a named reference inside an event body.
type EventRefInfo struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Line     int    `json:"line"`
	DefFile  string `json:"defFile,omitempty"`
	DefLine  int    `json:"defLine,omitempty"`
	DefCount int    `json:"defCount,omitempty"`
}

// EventDetail is the inspector payload for one event, including sim order.
type EventDetail struct {
	ID       string             `json:"id"`
	File     string             `json:"file"`
	Line     int                `json:"line"`
	EndLine  int                `json:"endLine"`
	Fields   []EventFieldInfo   `json:"fields"`
	Type     string             `json:"type,omitempty"`
	Hidden   bool               `json:"hidden,omitempty"`
	Theme    string             `json:"theme,omitempty"`
	Title    *EventLocField     `json:"title,omitempty"`
	Desc     *EventLocField     `json:"desc,omitempty"`
	Flavor   *EventLocField     `json:"flavor,omitempty"`
	Sections []EventSectionInfo `json:"sections"`
	Options  []EventOptionInfo  `json:"options"`
	Refs     []EventRefInfo     `json:"refs"`
	SimSteps []SimStep          `json:"simSteps"`
}

// SimStep is one firing-order block of an event.
type SimStep struct {
	Kind          string            `json:"kind"`
	Title         string            `json:"title"`
	Subtitle      string            `json:"subtitle"`
	Line          int               `json:"line"`
	Note          string            `json:"note"`
	Lines         []EventScriptLine `json:"lines"`
	Hidden        int               `json:"hidden"`
	Targets       []EventStepTarget `json:"targets"`
	HiddenTargets int               `json:"hiddenTargets"`
}

// LocIssue is one coverage finding.
type LocIssue struct {
	Key   string `json:"key"`
	File  string `json:"file,omitempty"`
	Line  int    `json:"line,omitempty"`
	Value string `json:"value,omitempty"`
}

// LocCoverage is per-language localization health for the workspace mods.
type LocCoverage struct {
	Language     string     `json:"language"`
	Defined      int        `json:"defined"`
	Missing      []LocIssue `json:"missing"`
	Orphaned     []LocIssue `json:"orphaned"`
	Untranslated []LocIssue `json:"untranslated"`
}

// LocLookup is one localization key's resolved english text and site.
type LocLookup struct {
	Key    string `json:"key"`
	Text   string `json:"text"`
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Origin string `json:"origin,omitempty"`
}

// DependencyItem is one named site in a dependency group.
type DependencyItem struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// DependencyGroup is dependents or dependencies of one kind.
type DependencyGroup struct {
	Kind  string           `json:"kind"`
	Items []DependencyItem `json:"items"`
}

// DependencyDef is the resolved subject of a dependency query.
type DependencyDef struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// Dependencies is the inspector payload for one definition.
type Dependencies struct {
	Def          *DependencyDef    `json:"def"`
	Dependents   []DependencyGroup `json:"dependents"`
	Dependencies []DependencyGroup `json:"dependencies"`
}

var (
	eventIDRe    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*\.\d+$`)
	targetNameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)
	scopePrefix  = regexp.MustCompile(`^(scope|var|local_var|global_var):([A-Za-z0-9_.-]+)$`)
)

func blockOf(v parser.Value) *parser.Block {
	switch b := v.(type) {
	case *parser.Block:
		return b
	case *parser.TaggedBlock:
		return &b.Block
	default:
		return nil
	}
}

func parseOf(s *session.Session, path string) parser.Result {
	src := s.FileText(path)
	if src == "" {
		return parser.Result{}
	}
	if r, ok := s.Result(path); ok && r.Src == src {
		return r
	}
	return parser.Parse(src)
}

func locValue(s *session.Session, key string) string {
	if idx := s.Index(); idx != nil {
		if v, ok := idx.Loc[key]; ok {
			return v
		}
	}
	if c := s.Cache(); c != nil && c.LocEnglish != nil {
		return c.LocEnglish[key]
	}
	return ""
}

func titleOf(s *session.Session, id string) string {
	keys := []string{
		id + ".t",
		strings.ReplaceAll(id, ".", "_") + "_t",
		id + ".title",
		id,
	}
	for _, key := range keys {
		if v := locValue(s, key); v != "" {
			return v
		}
	}
	return ""
}

func sourceOf(origin string) string {
	if origin == "" {
		return "vanilla"
	}
	return "mod"
}

// graphKind maps a folder-derived def type to event / on_action / decision, or "".
func graphKind(t string) string {
	switch t {
	case "event":
		return "event"
	case "on_action", "on_actions":
		return "on_action"
	case "decision", "decisions":
		return "decision"
	default:
		return ""
	}
}

func isEffectKind(t string) bool {
	return t == "scripted_effect" || t == "scripted_effects"
}

func isCallKind(t string) bool {
	switch t {
	case "scripted_effect", "scripted_effects",
		"scripted_trigger", "scripted_triggers",
		"scripted_modifier", "scripted_modifiers":
		return true
	default:
		return false
	}
}

func fireKind(key string) string {
	switch strings.ToLower(key) {
	case "trigger_event", "events", "random_events", "first_valid", "fallback":
		return "event"
	case "on_action", "on_actions":
		return "on_action"
	default:
		return ""
	}
}

func inFocus(s *session.Session, path, modRoot string) bool {
	origin, _, ok := s.Locate(path)
	if !ok {
		return false
	}
	if modRoot == "" {
		return true
	}
	for _, m := range s.Mods() {
		if m.Origin == origin {
			return m.Root == modRoot
		}
	}
	return false
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
