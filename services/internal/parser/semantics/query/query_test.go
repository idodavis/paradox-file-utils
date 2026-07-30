package query_test

import (
	"testing"

	"paradox-modding-tools/services/internal/parser/semantics/query"
)

func TestLoadEmbedded_CK3(t *testing.T) {
	m, err := query.LoadEmbedded("CK3")
	if err != nil {
		t.Fatalf("LoadEmbedded CK3: %v", err)
	}
	if m == nil {
		t.Fatalf("LoadEmbedded CK3: nil metadata")
	}
	if m.Game == "" {
		t.Fatalf("LoadEmbedded CK3: empty game")
	}
}
