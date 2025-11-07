package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"

	"paradox-file-utils/parser"
)

var (
	EVENT_KEY_PATTERN = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\.\d+$`)
	SYNC_OUTPUT_DIR   = "sync-output"
)

type Config struct {
	GameRoot            string `yaml:"game_root"`
	ModRoot             string `yaml:"mod_root"`
	CustomCommentPrefix string `yaml:"custom_comment_prefix"`
	Files               []struct {
		RelativePath       string   `yaml:"relative_path"`
		ModifiedEntityKeys []string `yaml:"modified_entity_keys"`
	} `yaml:"files"`
}

type ModdedObject struct {
	ObjectAsmt *parser.Assignment
	Comment    string
}

func loadConfig(path string) (*Config, error) {
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

func writeEntry(out *os.File, s string) {
	if _, err := out.WriteString(s); err != nil {
		fmt.Println("Error writing to output file:", err)
		out.Close()
		return
	}
}

// Goes through the parsed tree and finds block assignments,
// and if using normal event key syntax, adds them to a list
// Also grabs comments if they contain the specified prefix in the config
func collectObjectsAndComments(entries []*parser.Entry, prefix string) map[string]ModdedObject {
	events := make(map[string]ModdedObject)
	var pendingComments []string
	for _, entry := range entries {

		// Grabbing custom comments
		// These will be attached to the next event block found
		if c := entry.Comment; c != "" {
			if strings.HasPrefix(c, prefix) {
				pendingComments = append(pendingComments, c)
			}
			continue
		}

		// Grabbing events
		if asmt := entry.Assignment; asmt != nil {
			key := asmt.Key
			if asmt.Object != nil && EVENT_KEY_PATTERN.MatchString(key) {
				comment := ""
				if len(pendingComments) > 0 {
					comment = strings.Join(pendingComments, "\n") + "\n"
					pendingComments = nil
				}
				events[key] = ModdedObject{asmt, comment}
			}
		}
	}
	return events
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./vanilla_synchronizer <config.yaml>")
		return
	}
	cfg, err := loadConfig(os.Args[1])
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	for _, file := range cfg.Files {
		relPath := file.RelativePath

		// Using map to make quick key lookup/access
		modKeys := make(map[string]struct{})
		for _, k := range file.ModifiedEntityKeys {
			modKeys[k] = struct{}{}
		}

		// Parse vanilla file
		vanillaPath := filepath.Join(cfg.GameRoot, relPath)
		vanillaFile, err := parser.ParseFile(vanillaPath)
		if err != nil {
			fmt.Println("Error parsing vanilla file:", err)
			return
		}

		// Parse mod file
		modPath := filepath.Join(cfg.ModRoot, relPath)
		modFile, err := parser.ParseFile(modPath)
		if err != nil {
			fmt.Println("Error parsing mod file:", err)
			return
		}

		// Prepare output file
		outPath := SYNC_OUTPUT_DIR + "/" + filepath.Base(relPath)
		out, err := os.Create(outPath)
		if err != nil {
			fmt.Println("Error writing merged file:", err)
			return
		}

		// Collect events from mod file (all of them, not just ones that were actually modified)
		modFileObjects := collectObjectsAndComments(modFile.Entries, cfg.CustomCommentPrefix)

		// Walk vanilla lines, swap event blocks if needed
		for _, vEntry := range vanillaFile.Entries {
			if asmt := vEntry.Assignment; asmt != nil {
				key := asmt.Key

				// Swap event block if key is in modified event keys list
				if _, ok := modKeys[key]; ok {
					if modObject, ok := modFileObjects[key]; ok {
						// Write mod comment if exists
						if modObject.Comment != "" {
							writeEntry(out, modObject.Comment)
						}

						writeEntry(out, modObject.ObjectAsmt.GetRawText())
						continue
					}
				}
			}

			// Otherwise, write vanilla assignment as-is
			writeEntry(out, vEntry.GetRawText())
			continue

			// // Write vanilla comments as-is
			// if c := vEntry.Comment; c != "" {
			// 	writeEntry(out, c)
			// }
			// // Write vanilla whitespace as-is
			// if ws := vEntry.Whitespace; ws != "" {
			// 	writeEntry(out, ws)
			// }
		}
		fmt.Println("Merged file written to:", outPath)
		defer out.Close()
	}
}
