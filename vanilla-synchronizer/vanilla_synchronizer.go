package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"paradox-file-utils/parser"

	"github.com/antlr4-go/antlr/v4"
	"github.com/goccy/go-yaml"
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

type ModdedEvent struct {
	Event   *parser.AssignmentContext
	Comment string
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

func writeLine(out *os.File, s string) {
	if _, err := out.WriteString(s); err != nil {
		fmt.Println("Error writing to output file:", err)
		out.Close()
		return
	}
}

// Goes through the parsed tree and finds block assignments,
// and if using normal event key syntax, adds them to a list
// Also grabs comments if they contain the specified prefix in the config
func collectEventsAndComments(tree *parser.FileContext, prefix string) map[string]ModdedEvent {
	events := make(map[string]ModdedEvent)
	var pendingComments []string
	for _, line := range tree.AllLine() {

		// Grabbing custom comments
		// These will be attached to the next event block found
		if c := line.COMMENT(); c != nil {
			comment := c.GetText()
			if strings.HasPrefix(comment, prefix) {
				pendingComments = append(pendingComments, comment)
			}
			continue
		}

		// Grabbing events
		if stmt := line.Statement(); stmt != nil {
			if assign := stmt.Assignment(); assign != nil {
				key := assign.Key().GetText()
				if assign.Value().Block() != nil && EVENT_KEY_PATTERN.MatchString(key) {
					comment := ""
					if len(pendingComments) > 0 {
						comment = strings.Join(pendingComments, "\n") + "\n"
						pendingComments = nil
					}
					events[key] = ModdedEvent{assign.(*parser.AssignmentContext), comment}
				}
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
		vanillaInput, _ := antlr.NewFileStream(vanillaPath)
		vanillaLexer := parser.NewParadoxLexer(vanillaInput)
		vanillaStream := antlr.NewCommonTokenStream(vanillaLexer, antlr.TokenDefaultChannel)
		vanillaParser := parser.NewParadoxParser(vanillaStream)
		vanillaTree := vanillaParser.File()

		// Parse mod file
		modPath := filepath.Join(cfg.ModRoot, relPath)
		modInput, _ := antlr.NewFileStream(modPath)
		modLexer := parser.NewParadoxLexer(modInput)
		modStream := antlr.NewCommonTokenStream(modLexer, antlr.TokenDefaultChannel)
		modParser := parser.NewParadoxParser(modStream)
		modTree := modParser.File()

		// Prepare output file
		outPath := SYNC_OUTPUT_DIR + "/" + filepath.Base(relPath)
		out, err := os.Create(outPath)
		if err != nil {
			fmt.Println("Error writing merged file:", err)
			return
		}

		// Collect events from mod file (all of them, not just ones that were actually modified)
		modFileEvents := collectEventsAndComments(modTree.(*parser.FileContext), cfg.CustomCommentPrefix)

		// Walk vanilla lines, swap event blocks if needed
		for _, line := range vanillaTree.AllLine() {
			if stmt := line.Statement(); stmt != nil {
				if assign := stmt.Assignment(); assign != nil {
					key := assign.Key().GetText()

					// Swap event block if key is in modified event keys list
					if _, ok := modKeys[key]; ok {
						if modEvent, ok := modFileEvents[key]; ok {
							// Write mod comment if exists
							if modEvent.Comment != "" {
								writeLine(out, modEvent.Comment)
							}

							writeLine(out, modEvent.Event.GetText())
							continue
						}
					}

					// Otherwise, write vanilla assignment as-is
					writeLine(out, assign.GetText())
					continue
				}
			}

			// Write vanilla comments as-is
			if c := line.COMMENT(); c != nil {
				writeLine(out, c.GetText())
			}
			// Write vanilla whitespace as-is
			if ws := line.WS(); ws != nil {
				writeLine(out, ws.GetText())
			}
		}
		fmt.Println("Merged file written to:", outPath)
		defer out.Close()
	}
}
