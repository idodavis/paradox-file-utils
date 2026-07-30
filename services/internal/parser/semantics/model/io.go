package model

import (
	"encoding/json"
	"os"
)

func Load(path string) (*SemanticMetadata, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Decode(b)
}

func Save(path string, m *SemanticMetadata) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func Decode(b []byte) (*SemanticMetadata, error) {
	var m SemanticMetadata
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
