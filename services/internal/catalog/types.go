// types.go declares persisted semantic-model shapes and format-version constants.
// A version bump invalidates on-disk files (no migration).

package catalog

import "strings"

// CacheFormatVersion is the on-disk schema of VanillaCache.
const CacheFormatVersion = 14

// LocFormatVersion is the on-disk schema of a vanilla loc sidecar.
const LocFormatVersion = 3

// Def is one named object: Kind/Key, absolute Path, 0-based Line, Origin ("" = vanilla).
// OwnerKey is the enclosing scripted macro for script_param defs.
// Value is the harvested RHS literal at a ScriptName def site (hover body).
type Def struct {
	Kind     string `json:"kind"`
	Key      string `json:"key"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Start    int    `json:"start,omitempty"`
	End      int    `json:"end,omitempty"`
	Origin   string `json:"origin,omitempty"`
	OwnerKey string `json:"ownerKey,omitempty"`
	Value    string `json:"value,omitempty"`
}

// Ref is a use-site. Kind is "loc", "loc-broad", "loc-convention",
// "event", "on_action", or an ephemeral/script kind. OwnerKey scopes
// script_param refs to the enclosing scripted_* call.
type Ref struct {
	Key      string `json:"key"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	OwnerKey string `json:"ownerKey,omitempty"`
}

// Edge is a directed event-graph link. Kind "call" is a scripted-effect invocation.
type Edge struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Via     string `json:"via"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Kind    string `json:"kind,omitempty"`
	NameKey string `json:"nameKey,omitempty"`
}

// ModInput is one workspace mod: Origin id, Root folder, load Order, display Name.
type ModInput struct {
	Origin string
	Root   string
	Order  int
	Name   string
}

// LocEntry is one localization value with its source file and line.
type LocEntry struct {
	Value string `json:"value"`
	Path  string `json:"path"`
	Line  int    `json:"line"`
}

// LocDelta is the loc payload one file contributes to the harvest.
type LocDelta struct {
	Lang string
	Vals map[string]LocEntry
}

// VanillaCache is the vanilla script model for one install + version.
type VanillaCache struct {
	FormatVersion int    `json:"formatVersion"`
	InstallID     string `json:"installId"`
	GameID        string `json:"gameId"`
	InstallPath   string `json:"installPath"`
	GameVersion   string `json:"gameVersion"`
	ScannedAt     string `json:"scannedAt"`

	Defs             []Def                          `json:"defs"`
	FieldInfo        map[string]string              `json:"fieldInfo"`
	FieldInfoByKind  map[string]map[string]string   `json:"fieldInfoByKind"`
	Structures       map[string][]string            `json:"structures"`
	StructureBlocks  map[string][]string            `json:"structureBlocks,omitempty"`
	Vocabulary       []string                       `json:"vocabulary"`
	Effects          []string                       `json:"effects"`
	Triggers         []string                       `json:"triggers"`
	GUITypes         []string                       `json:"guiTypes"`
	GUIProps         []string                       `json:"guiProps"`
	MetaKeys         []string                       `json:"metaKeys"`
	Edges            []Edge                         `json:"edges,omitempty"`
	LocRefs          []Ref                          `json:"locRefs,omitempty"`
	FieldValueKinds  map[string]string              `json:"fieldValueKinds,omitempty"`
	FieldEnumsByKind map[string]map[string][]string `json:"fieldEnumsByKind,omitempty"`
	TokenUsage       map[string]string              `json:"tokenUsage,omitempty"`
	TokenDoc         map[string]string              `json:"tokenDoc,omitempty"`
	TokenScopes      map[string]string              `json:"tokenScopes,omitempty"`
	DataFunctions    []string                       `json:"dataFunctions,omitempty"`

	// Built by PrepareCache on load/scan; not persisted.
	effectSet          map[string]bool            `json:"-"`
	triggerSet         map[string]bool            `json:"-"`
	vocabSet           map[string]bool            `json:"-"`
	guiTypesSet        map[string]bool            `json:"-"`
	guiPropsSet        map[string]bool            `json:"-"`
	metaKeysSet        map[string]bool            `json:"-"`
	structureSets      map[string]map[string]bool `json:"-"`
	structureBlockSets map[string]map[string]bool `json:"-"`
}

// VanillaLoc is the default-language loc sidecar for one install + version + lang.
type VanillaLoc struct {
	FormatVersion int                 `json:"formatVersion"`
	Sites         map[string]LocEntry `json:"sites"`
}

// PrepareCache normalizes field docs and builds O(1) membership maps for LSP/views.
func PrepareCache(c *VanillaCache) {
	if c == nil {
		return
	}
	if c.FieldInfo == nil {
		c.FieldInfo = map[string]string{}
	}
	if c.FieldInfoByKind == nil {
		c.FieldInfoByKind = map[string]map[string]string{}
	}
	if c.TokenDoc == nil {
		c.TokenDoc = map[string]string{}
	}
	if c.Structures == nil {
		c.Structures = map[string][]string{}
	}
	if c.StructureBlocks == nil {
		c.StructureBlocks = map[string][]string{}
	}
	if c.FieldValueKinds == nil {
		c.FieldValueKinds = map[string]string{}
	}
	if c.FieldEnumsByKind == nil {
		c.FieldEnumsByKind = map[string]map[string][]string{}
	}
	c.FieldInfo = normalizeDocMap(c.FieldInfo)
	c.TokenDoc = normalizeDocMap(c.TokenDoc)
	for kind, m := range c.FieldInfoByKind {
		c.FieldInfoByKind[kind] = normalizeDocMap(m)
	}
	c.effectSet, c.triggerSet = sliceSet(c.Effects), sliceSet(c.Triggers)
	c.vocabSet, c.guiTypesSet = sliceSet(c.Vocabulary), sliceSet(c.GUITypes)
	c.guiPropsSet, c.metaKeysSet = sliceSet(c.GUIProps), sliceSet(c.MetaKeys)
	c.structureSets = map[string]map[string]bool{}
	for kind, keys := range c.Structures {
		c.structureSets[kind] = sliceSet(keys)
	}
	c.structureBlockSets = map[string]map[string]bool{}
	for kind, keys := range c.StructureBlocks {
		c.structureBlockSets[kind] = sliceSet(keys)
	}
}

// StructureBlock reports whether key is usually a block under kind.
func (c *VanillaCache) StructureBlock(kind, key string) bool {
	if c == nil || key == "" {
		return false
	}
	set := c.structureBlockSets[kind]
	return set[key] || set[strings.ToLower(key)]
}

// MemberSets returns pre-built structure/effect/trigger maps for LSP rank.
func (c *VanillaCache) MemberSets(kind string) (structKeys, effects, triggers map[string]bool) {
	if c == nil {
		return nil, nil, nil
	}
	return c.structureSets[kind], c.effectSet, c.triggerSet
}

func sliceSet(keys []string) map[string]bool {
	if len(keys) == 0 {
		return nil
	}
	m := make(map[string]bool, len(keys)*2)
	for _, k := range keys {
		m[k] = true
		m[strings.ToLower(k)] = true
	}
	return m
}

func normalizeDocMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return m
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		lk := strings.ToLower(k)
		if out[lk] == "" {
			out[lk] = v
		}
	}
	return out
}
