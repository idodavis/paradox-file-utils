package config

import (
	"os"

	"paradox-file-utils/internal/core"

	"github.com/goccy/go-yaml"
)

// Config represents the configuration for the synchronizer
type Config struct {
	GameRoot            string `yaml:"game_root"`
	ModRoot             string `yaml:"mod_root"`
	CustomCommentPrefix string `yaml:"custom_comment_prefix"`
	// PrecedenceMode is ignored - A always wins. Users can swap directories to change which is base.
	PrecedenceMode string `yaml:"precedence_mode,omitempty"`
	Files          []struct {
		RelativePath       string   `yaml:"relative_path"`
		ModifiedEntityKeys []string `yaml:"modified_entity_keys"`
		// Per-file precedence override
		PrecedenceMode string `yaml:"precedence_mode,omitempty"`
	} `yaml:"files"`
}

// LoadConfig loads a configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ToMergeOptions converts a Config to MergeOptions for a specific file
// Note: PrecedenceMode is ignored - A always wins. Users can swap directories to change which is base.
func (c *Config) ToMergeOptions(fileIndex int) core.MergeOptions {
	if fileIndex < 0 || fileIndex >= len(c.Files) {
		return core.MergeOptions{
			AddAdditionalEntries: true,
			EntryPlacement:       "bottom",
			KeyList:              []string{},
			CustomCommentPrefix:  c.CustomCommentPrefix,
		}
	}

	file := c.Files[fileIndex]

	return core.MergeOptions{
		AddAdditionalEntries: true,
		EntryPlacement:       "bottom",
		KeyList:              file.ModifiedEntityKeys,
		CustomCommentPrefix:  c.CustomCommentPrefix,
	}
}
