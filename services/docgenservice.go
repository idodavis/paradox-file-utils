package services

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"

	internal "paradox-modding-tools/services/internal"
	semanticquery "paradox-modding-tools/services/internal/parser/semantics/query"
	"paradox-modding-tools/services/internal/repos"

	"github.com/jmoiron/sqlx"
)

// DocGenService generates semantic-aware docs from script files.
type DocGenService struct {
	FileService *FileService
	DB          *sqlx.DB
	repo        *repos.ModDocRepository
}

func (d *DocGenService) getRepo() *repos.ModDocRepository {
	if d.repo == nil {
		d.repo = repos.NewModDocRepository(d.DB)
	}
	return d.repo
}

func (d *DocGenService) docGame(game string) string {
	return strings.ToLower(strings.TrimSpace(game))
}

// ScanSemantic builds semantic doc records from script files via parser+semantic query flow.
func (d *DocGenService) ScanSemantic(game, installPath string, objectTypes []string) ([]string, error) {
	gameNorm := d.docGame(game)
	installPath = strings.TrimSpace(installPath)
	if gameNorm == "" || installPath == "" {
		return nil, errors.New("game and install path required")
	}

	gameScriptRoot, err := d.FileService.GetGameScriptRoot(game, installPath)
	if err != nil {
		return nil, err
	}
	files, err := d.FileService.CollectFilesFromPath(gameScriptRoot, FileCollectorFilter{Extensions: []string{".txt"}})
	if err != nil {
		return nil, err
	}

	hash := internal.InstallPathHash(installPath)
	fetchedAt := time.Now().UTC().Format(time.RFC3339)
	repo := d.getRepo()
	out := make([]string, 0, len(files))

	for relPath, absPath := range files {
		entities, err := semanticquery.ExtractEntityRecordsFromFile(absPath, game, objectTypes)
		if err != nil || len(entities) == 0 {
			continue
		}
		semanticquery.EnrichEntityRelationships(entities)
		payload, err := json.MarshalIndent(map[string]any{
			"game":     strings.ToUpper(gameNorm),
			"file":     filepath.ToSlash(relPath),
			"entities": entities,
		}, "", "  ")
		if err != nil {
			continue
		}
		docRelPath := filepath.ToSlash(relPath) + ".semantic.json"
		if err := repo.UpsertDocFile(gameNorm, hash, docRelPath, absPath, string(payload), fetchedAt); err != nil {
			return nil, err
		}
		out = append(out, docRelPath)
	}

	return out, nil
}
