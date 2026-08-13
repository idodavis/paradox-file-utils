// Package schemaeval is a JSON-driven Paradox object-schema classifier shared by games.
package schemaeval

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Schema defines how to identify a type of script object.
type Schema struct {
	Name              string   `json:"name"`
	Paths             []string `json:"paths"`
	KeyPattern        string   `json:"keyPattern"`
	KeyPrefixes       []string `json:"keyPrefixes,omitempty"`
	KeySuffixes       []string `json:"keySuffixes,omitempty"`
	Attributes        []string `json:"attributes,omitempty"`
	InlineKeyPattern  string   `json:"inlineKeyPattern,omitempty"`
	InlineKeyKeywords []string `json:"inlineKeyKeywords,omitempty"`
}

type schemaFile struct {
	Schemas map[string]Schema `json:"schemas"`
}

type matcherParams struct {
	Keywords, Prefixes, Suffixes []string
}

// Pack is a loaded per-game schema set.
type Pack struct {
	schemas  map[string]Schema
	matchers map[string]func(key string, p matcherParams) (string, bool)
}

var (
	namespacedRE = regexp.MustCompile(`^[\w]+\.[\w.]+$`)
	identifierRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	dateRE       = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

// MustLoad parses schema JSON or panics.
func MustLoad(jsonBytes []byte, label string) *Pack {
	p, err := Load(jsonBytes)
	if err != nil {
		panic(fmt.Sprintf("%s: %v", label, err))
	}
	return p
}

// Load parses schema JSON into a Pack.
func Load(jsonBytes []byte) (*Pack, error) {
	var f schemaFile
	if err := json.Unmarshal(jsonBytes, &f); err != nil {
		return nil, fmt.Errorf("parse schemas: %w", err)
	}
	p := &Pack{
		schemas:  f.Schemas,
		matchers: defaultMatchers(),
	}
	return p, nil
}

func defaultMatchers() map[string]func(key string, p matcherParams) (string, bool) {
	return map[string]func(key string, p matcherParams) (string, bool){
		"numeric": func(key string, _ matcherParams) (string, bool) {
			_, err := strconv.ParseInt(key, 10, 64)
			return key, err == nil
		},
		"prefixed": func(key string, p matcherParams) (string, bool) {
			for _, pre := range p.Prefixes {
				if strings.HasPrefix(key, pre) {
					return key, true
				}
			}
			return "", false
		},
		"suffixed": func(key string, p matcherParams) (string, bool) {
			for _, suf := range p.Suffixes {
				if strings.HasSuffix(key, suf) {
					return key, true
				}
			}
			return "", false
		},
		"namespaced": func(key string, _ matcherParams) (string, bool) {
			return key, namespacedRE.MatchString(key)
		},
		"keyword_prefixed": matcherKeywordPrefixed,
		"identifier_no_dot": func(key string, _ matcherParams) (string, bool) {
			return key, identifierRE.MatchString(key)
		},
		"date": func(key string, _ matcherParams) (string, bool) {
			return key, dateRE.MatchString(key)
		},
		"any": func(key string, _ matcherParams) (string, bool) { return key, true },
	}
}

func matcherKeywordPrefixed(key string, p matcherParams) (string, bool) {
	for _, kw := range p.Keywords {
		if (strings.HasPrefix(key, kw+" ") || (len(key) > len(kw) && strings.HasPrefix(key, kw))) && len(key) > len(kw) {
			displayKey := strings.TrimPrefix(key, kw+" ")
			if displayKey == key {
				displayKey = strings.TrimPrefix(key, kw)
			}
			if displayKey != "" {
				return displayKey, true
			}
		}
	}
	return "", false
}

// GetSchema returns the schema for a type name.
func (p *Pack) GetSchema(typeName string) (Schema, bool) {
	s, ok := p.schemas[typeName]
	return s, ok
}

// GetSchemaNames returns all object type names.
func (p *Pack) GetSchemaNames() []string {
	names := make([]string, 0, len(p.schemas))
	for n := range p.schemas {
		names = append(names, n)
	}
	return names
}

// ApplicableTypesForPath returns type names whose schema paths match filePath.
func (p *Pack) ApplicableTypesForPath(filePath string) []string {
	norm := filepath.ToSlash(filepath.Clean(filePath))
	pSlash := func(path string) string { return filepath.ToSlash(path) }
	var out []string
	for name, s := range p.schemas {
		for _, path := range s.Paths {
			q := pSlash(path)
			if norm == q || strings.HasPrefix(norm, q+"/") {
				out = append(out, name)
				break
			}
			if strings.Contains(norm, "/"+q+"/") || strings.HasSuffix(norm, "/"+q) {
				out = append(out, name)
				break
			}
		}
	}
	return out
}

// MatchKey returns (displayKey, true) if key matches the schema pattern.
func (p *Pack) MatchKey(key string, schema *Schema, inline bool) (displayKey string, ok bool) {
	pattern := schema.KeyPattern
	params := matcherParams{schema.InlineKeyKeywords, schema.KeyPrefixes, schema.KeySuffixes}
	if inline && schema.InlineKeyPattern != "" {
		pattern = schema.InlineKeyPattern
	}
	fn, ok := p.matchers[pattern]
	if !ok {
		return "", false
	}
	displayKey, ok = fn(key, params)
	if !ok {
		return "", false
	}
	if schema.KeyPattern == "keyword_prefixed" {
		displayKey = strings.TrimSpace(displayKey)
		if displayKey == key || displayKey == "" || !identifierRE.MatchString(displayKey) {
			return "", false
		}
	}
	for _, kw := range schema.InlineKeyKeywords {
		if displayKey == kw {
			return "", false
		}
	}
	return displayKey, true
}

func (p *Pack) otherInline(key, excludeType string) bool {
	for name, s := range p.schemas {
		if name == excludeType || s.InlineKeyPattern == "" {
			continue
		}
		if _, ok := p.MatchKey(key, &s, true); ok {
			return true
		}
	}
	return false
}

// ClassifyKey returns (typeName, displayKey, true) for the best matching applicable type.
func (p *Pack) ClassifyKey(
	key string,
	hasObject bool,
	attrs map[string]bool,
	applicableTypes []string,
	inline bool,
) (typeName, displayKey string, ok bool) {
	var candidates []struct {
		t          string
		displayKey string
		score      int
	}
	for _, t := range applicableTypes {
		schema, has := p.GetSchema(t)
		if !has || p.otherInline(key, t) {
			continue
		}
		_, inlineMatch := p.MatchKey(key, &schema, true)
		needObj := schema.KeyPattern == "keyword_prefixed" ||
			(schema.InlineKeyPattern == "keyword_prefixed" && inlineMatch)
		if needObj && !hasObject {
			continue
		}
		displayKey, matchOk := p.MatchKey(key, &schema, inline)
		if !matchOk {
			if schema.InlineKeyPattern == "keyword_prefixed" && inlineMatch {
				displayKey, _ = p.MatchKey(key, &schema, true)
			}
			if displayKey == "" {
				continue
			}
		}
		score := 0
		if len(attrs) > 0 {
			for _, a := range schema.Attributes {
				if attrs[a] {
					score++
				}
			}
		}
		candidates = append(candidates, struct {
			t          string
			displayKey string
			score      int
		}{t, displayKey, score})
	}
	if len(candidates) == 0 {
		return "", "", false
	}
	best := 0
	for i := 1; i < len(candidates); i++ {
		if candidates[i].score > candidates[best].score {
			best = i
		}
	}
	return candidates[best].t, candidates[best].displayKey, true
}

// InlineTypesFor returns type names that have InlineKeyPattern set.
func (p *Pack) InlineTypesFor(objectTypes []string) []string {
	var out []string
	for _, t := range objectTypes {
		s, ok := p.GetSchema(t)
		if ok && s.InlineKeyPattern != "" {
			out = append(out, t)
		}
	}
	return out
}

// TypeForPath returns the first schema type name applicable to path, or "".
func (p *Pack) TypeForPath(filePath string) string {
	types := p.ApplicableTypesForPath(filePath)
	if len(types) == 0 {
		return ""
	}
	return types[0]
}
