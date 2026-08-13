// Package semantics: packs.go selects the effective schema pack per game.
package semantics

import (
	_ "embed"

	"paradox-modding-tools/services/internal/semantics/schemaeval"
)

//go:embed bootstrap/ck3.json
var ck3BootstrapJSON []byte

//go:embed bootstrap/eu5.json
var eu5BootstrapJSON []byte

//go:embed bootstrap/vic3.json
var vic3BootstrapJSON []byte

var (
	bootstraps = map[string]*Bootstrap{}
	packs      = map[string]*schemaeval.Pack{}
)

func init() {
	load := func(id string, raw []byte) {
		b, err := ParseBootstrap(raw)
		if err != nil {
			panic(err)
		}
		bootstraps[id] = b
		packs[id] = PackFromBootstrap(b)
	}
	load("ck3", ck3BootstrapJSON)
	load("eu5", eu5BootstrapJSON)
	load("vic3", vic3BootstrapJSON)
}

// BootstrapFor returns committed bootstrap for a game id, or nil.
func BootstrapFor(gameID string) *Bootstrap {
	return bootstraps[gameID]
}

// ForGame returns the schema pack for a game id (bootstrap; cache merge later).
func ForGame(gameID string) *schemaeval.Pack {
	return packs[gameID]
}

// SetPackForGame replaces the effective pack (used after install cache merge).
func SetPackForGame(gameID string, p *schemaeval.Pack) {
	if p != nil {
		packs[gameID] = p
	}
}
