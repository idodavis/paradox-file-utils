package query

import (
	"fmt"
	"sort"

	pdxparser "paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/parser/semantics/model"
)

func Load(path string) (*model.SemanticMetadata, error) {
	return model.Load(path)
}

func LoadEmbedded(game string) (*model.SemanticMetadata, error) {
	b, err := pdxparser.SemanticMetadataBytes(game)
	if err != nil {
		return nil, err
	}
	return model.Decode(b)
}

func MustLoadEmbedded(game string) *model.SemanticMetadata {
	m, err := LoadEmbedded(game)
	if err != nil {
		panic(err)
	}
	return m
}

func SupportedTypes(m *model.SemanticMetadata) []string {
	if m == nil || len(m.Types) == 0 {
		return nil
	}
	names := make([]string, 0, len(m.Types))
	for n := range m.Types {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func AttributesForType(m *model.SemanticMetadata, typeName string) ([]string, error) {
	if m == nil {
		return nil, fmt.Errorf("query: nil semantic metadata")
	}
	spec, ok := m.Types[typeName]
	if !ok {
		return nil, nil
	}
	if len(spec.AttributeNames) == 0 {
		return nil, nil
	}
	attrs := append([]string(nil), spec.AttributeNames...)
	sort.Strings(attrs)
	return attrs, nil
}
