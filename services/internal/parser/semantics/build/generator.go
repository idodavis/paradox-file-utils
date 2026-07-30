package build

import (
	"errors"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/parser/semantics/model"
)

type parsedTxt struct {
	relPath string
	absPath string
	tree    *parser.ParadoxFile
}

// Generate walks gameRoot for .txt files, runs parser + AST passes, and writes games/{game}/semantic-metadata.json.
func Generate(gameRoot, game string) error {
	out, err := defaultSemanticOutputPath(game)
	if err != nil {
		return err
	}
	return generateTo(gameRoot, game, out)
}

func defaultSemanticOutputPath(game string) (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("build: runtime.Caller failed")
	}
	g := strings.ToLower(strings.TrimSpace(game))
	return filepath.Join(filepath.Dir(file), "..", "..", "games", g, "semantic-metadata.json"), nil
}

func generateTo(gameRoot, game, outputPath string) error {
	g := strings.ToLower(strings.TrimSpace(game))
	meta := &model.SemanticMetadata{
		Game:          strings.ToUpper(g),
		Version:       1,
		GeneratedAt:   time.Now().UTC(),
		LanguageFacts: map[string]any{},
		Types:         map[string]model.TypeSpec{},
		Diagnostics:   []model.Diagnostic{},
	}

	var parsed []parsedTxt
	err := filepath.WalkDir(gameRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".txt" {
			return nil
		}
		rel, relErr := filepath.Rel(gameRoot, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		f, perr := parser.ParseFile(path)
		if perr != nil {
			meta.Diagnostics = append(meta.Diagnostics, model.Diagnostic{Path: path, Message: perr.Error()})
			return nil
		}
		parsed = append(parsed, parsedTxt{relPath: rel, absPath: path, tree: f})
		return nil
	})
	if err != nil {
		return err
	}
	if err := runPasses(meta, g, parsed); err != nil {
		return err
	}
	return model.Save(outputPath, meta)
}
