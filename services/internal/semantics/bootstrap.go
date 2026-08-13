// Package semantics loads per-game bootstrap schemas and install semantic caches.
package semantics

import (
	"encoding/json"
	"fmt"

	"paradox-modding-tools/services/internal/semantics/schemaeval"
)

// Bootstrap is thin per-game config committed in-repo (roots + path stubs).
type Bootstrap struct {
	ScriptRoot       string                      `json:"scriptRoot"`
	GuiRoots         []string                    `json:"guiRoots"`
	LocRoots         []string                    `json:"locRoots"`
	DocumentsFolder  string                      `json:"documentsFolder"`
	ScriptDocsRel    string                      `json:"scriptDocsRel"`
	CommonDirs       []string                    `json:"commonDirs"`
	LocAttrSeed      []string                    `json:"locAttrSeed"`
	InfoHints        []string                    `json:"infoHints"`
	Schemas          map[string]schemaeval.Schema `json:"schemas"`
}

// ParseBootstrap unmarshals bootstrap JSON bytes.
func ParseBootstrap(data []byte) (*Bootstrap, error) {
	var b Bootstrap
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parse bootstrap: %w", err)
	}
	if b.Schemas == nil {
		b.Schemas = map[string]schemaeval.Schema{}
	}
	return &b, nil
}

// PackFromBootstrap builds a schemaeval Pack from bootstrap schemas.
func PackFromBootstrap(b *Bootstrap) *schemaeval.Pack {
	raw, err := json.Marshal(struct {
		Schemas map[string]schemaeval.Schema `json:"schemas"`
	}{Schemas: b.Schemas})
	if err != nil {
		panic(err)
	}
	return schemaeval.MustLoad(raw, "bootstrap")
}

// LocAttrSeed returns bootstrap loc-reference attribute names, or defaults.
func (b *Bootstrap) LocAttrs() []string {
	if b == nil || len(b.LocAttrSeed) == 0 {
		return []string{"title", "desc", "description", "name", "text", "tooltip"}
	}
	return b.LocAttrSeed
}
