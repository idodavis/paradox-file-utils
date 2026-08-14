// Package langmodel builds and caches per-workspace script definition/edge models.
package langmodel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const appConfigDirName = "Paradox Modding Tools"

// Def is a top-level script definition extracted from a workspace file.
type Def struct {
	Type     string `json:"type"`
	Key      string `json:"key"`
	FilePath string `json:"filePath"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	EndLine  int    `json:"endLine"`
	Summary  string `json:"summary"`
}

// Edge links a definition key to a referenced key.
type Edge struct {
	FromKey  string `json:"fromKey"`
	ToKey    string `json:"toKey"`
	EdgeType string `json:"edgeType"`
}

// Model is the workspace language model persisted as JSON.
type Model struct {
	WorkspaceID string `json:"workspaceId"`
	BuiltAt     string `json:"builtAt"`
	Definitions []Def  `json:"definitions"`
	Edges       []Edge `json:"edges"`
}

// ModelDir returns the user-data directory for language models.
func ModelDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, appConfigDirName, "langmodels")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// ModelPath returns the JSON path for a workspace model.
func ModelPath(workspaceID string) (string, error) {
	dir, err := ModelDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, workspaceID+".json"), nil
}

// Save writes the model to user-data.
func Save(m *Model) error {
	if m == nil || m.WorkspaceID == "" {
		return fmt.Errorf("model workspace id is required")
	}
	if m.BuiltAt == "" {
		m.BuiltAt = time.Now().UTC().Format(time.RFC3339)
	}
	path, err := ModelPath(m.WorkspaceID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load reads a previously saved workspace model.
func Load(workspaceID string) (*Model, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	path, err := ModelPath(workspaceID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Status reports whether a cached model exists.
type Status struct {
	Present     bool   `json:"present"`
	WorkspaceID string `json:"workspaceId"`
	BuiltAt     string `json:"builtAt,omitempty"`
	DefCount    int    `json:"defCount"`
	EdgeCount   int    `json:"edgeCount"`
	Path        string `json:"path,omitempty"`
}

// GetStatus returns cache presence for a workspace.
func GetStatus(workspaceID string) (*Status, error) {
	st := &Status{WorkspaceID: workspaceID}
	path, err := ModelPath(workspaceID)
	if err != nil {
		return st, nil
	}
	st.Path = path
	m, err := Load(workspaceID)
	if err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			return st, nil
		}
		return nil, err
	}
	st.Present = true
	st.BuiltAt = m.BuiltAt
	st.DefCount = len(m.Definitions)
	st.EdgeCount = len(m.Edges)
	return st, nil
}
