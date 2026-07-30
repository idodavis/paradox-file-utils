package model

import (
	"time"
)

// SemanticMetadata is the on-disk shape for games/{game}/semantic-metadata.json.
type SemanticMetadata struct {
	Game          string              `json:"game"`
	Version       int                 `json:"version"`
	GeneratedAt   time.Time           `json:"generated_at"`
	LanguageFacts map[string]any      `json:"language_facts"`
	Types         map[string]TypeSpec `json:"types"`
	Diagnostics   []Diagnostic        `json:"diagnostics"`
}

// TypeSpec describes one script object/entity kind at a structural level.
type TypeSpec struct {
	Paths              []string         `json:"paths,omitempty"`
	KeyPattern         string           `json:"key_pattern,omitempty"`
	InlineKeyPattern   string           `json:"inline_key_pattern,omitempty"`
	ContainmentPattern string           `json:"containment_pattern,omitempty"`
	AttributeNames     []string         `json:"attribute_names,omitempty"`
	UsageForms         map[string]int64 `json:"usage_forms,omitempty"`
	ObservedFolders    []string         `json:"observed_folders,omitempty"`
	ObservedFiles      []string         `json:"observed_files,omitempty"`
	AssociatedInfo     []string         `json:"associated_info,omitempty"`
}

// Diagnostic is a build or parse warning attached to output metadata.
type Diagnostic struct {
	Path    string `json:"path,omitempty"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}
