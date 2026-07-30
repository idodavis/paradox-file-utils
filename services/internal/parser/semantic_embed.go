package parser

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed games/ck3/semantic-metadata.json games/eu5/semantic-metadata.json
var semanticMetadataFS embed.FS

// SemanticMetadataBytes returns embedded semantic-metadata.json for game (e.g. "CK3", "EU5").
func SemanticMetadataBytes(game string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(game)) {
	case "":
		return nil, fmt.Errorf("parser: empty game")
	case "ck3":
		return semanticMetadataFS.ReadFile("games/ck3/semantic-metadata.json")
	case "eu5":
		return semanticMetadataFS.ReadFile("games/eu5/semantic-metadata.json")
	default:
		return nil, fmt.Errorf("parser: unknown game %q", game)
	}
}
