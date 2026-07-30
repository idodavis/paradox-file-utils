package services

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	semanticquery "paradox-modding-tools/services/internal/parser/semantics/query"
	"paradox-modding-tools/services/internal/repos"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
)

var blackListPaths = []string{
	"content_source",
	"common/console_groups",
	"fonts",
	"gfx",
	"licenses",
	"sound",
	"tests",
	"tools",
	"map_data",
	"gui",
	"reader_export",
	"dlc",
	"dlc_metadata",
	"data_binding",
}

var maxConcurrency = max(1, int(float64(runtime.GOMAXPROCS(0))*0.80))

// InventoryService exposes inventory functionality to the Wails frontend (supported types, schema, extraction, save, list).
type InventoryService struct {
	DB   *sqlx.DB
	repo *repos.InventoryRepository
}

type (
	InventorySummary = repos.InventorySummary
	InventoryItemRow = repos.InventoryItemRow
	ItemDetails      = repos.ItemDetails
)

func (i *InventoryService) getRepo() *repos.InventoryRepository {
	if i.repo == nil {
		i.repo = repos.NewInventoryRepository(i.DB)
	}
	return i.repo
}

// GetSupportedTypes returns the sorted list of object type names for the given game.
func (i *InventoryService) GetSupportedTypes(game string) ([]string, error) {
	if strings.TrimSpace(game) == "" {
		return nil, nil
	}
	meta, err := semanticquery.LoadEmbedded(game)
	if err != nil {
		return nil, err
	}
	return semanticquery.SupportedTypes(meta), nil
}

// GetAttributes returns the list of attribute names for an object type and game.
func (i *InventoryService) GetAttributes(game, typeName string) ([]string, error) {
	if strings.TrimSpace(game) == "" || strings.TrimSpace(typeName) == "" {
		return nil, nil
	}
	meta, err := semanticquery.LoadEmbedded(game)
	if err != nil {
		return nil, err
	}
	return semanticquery.AttributesForType(meta, typeName)
}

// ExtractInventory extracts inventory items from basePath that match objectTypes. Writes to DB as temporary; returns inventoryId and totalCount. Only .txt files are processed. Deletes prior temporary inventories before inserting.
func (i *InventoryService) ExtractInventory(ctx context.Context, game, basePath string, objectTypes []string) (*string, error) {
	// Wipe any previous temporary inventories to keep the DB clean
	_ = i.getRepo().DeleteTemporaryInventories()

	shouldSkip := func(path string) bool {
		rel, _ := filepath.Rel(basePath, path)
		return slices.Contains(blackListPaths, filepath.ToSlash(rel))
	}

	semaphore := make(chan struct{}, maxConcurrency)
	eg, egCtx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	var items []repos.InventoryItem

	walkErr := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkip(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".txt" {
			return nil
		}

		filePath := path
		eg.Go(func() error {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if egCtx.Err() != nil {
				return egCtx.Err()
			}

			itemsForFile, err := semanticquery.ExtractInventoryItemsFromFile(filePath, game, objectTypes)
			if err != nil {
				log.Printf("inventory: skip (parse error) %s: %v", filePath, err)
				return nil
			}

			mu.Lock()
			for _, it := range itemsForFile {
				items = append(items, toRepoInventoryItem(it))
			}
			mu.Unlock()

			return nil
		})
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	if walkErr != nil && walkErr != fs.SkipAll {
		return nil, fmt.Errorf("walk %s: %w", basePath, walkErr)
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	queryItems := make([]semanticquery.InventoryItem, 0, len(items))
	for _, it := range items {
		queryItems = append(queryItems, toQueryInventoryItem(it))
	}
	semanticquery.EnrichInventoryReferences(queryItems)
	for idx := range queryItems {
		items[idx].References = toRepoRefs(queryItems[idx].References)
		items[idx].Referrers = toRepoRefs(queryItems[idx].Referrers)
	}

	repo := i.getRepo()

	inventoryId := uuid.New().String()
	name := fmt.Sprintf("%s - %s", game, time.Now().Format("2006-01-02"))
	if err := repo.CreateInventory(inventoryId, name, game, basePath, objectTypes); err != nil {
		return nil, fmt.Errorf("insert inventory: %w", err)
	}

	if err := repo.SaveInventoryItems(inventoryId, items); err != nil {
		return nil, err
	}

	return &inventoryId, nil
}

// ListInventoriesForGame returns saved (non-temporary) inventories for the given game, ordered by created_at DESC.
func (i *InventoryService) ListInventoriesForGame(game string) ([]InventorySummary, error) {
	return i.getRepo().ListInventories(game)
}

// SaveInventory marks an inventory as saved (persists name and sets is_temporary=0).
func (i *InventoryService) SaveInventory(id, name string) error {
	return i.getRepo().SaveInventory(id, name)
}

// GetInventoryItems returns lightweight rows for the grid (no raw_text, references, referrers).
func (i *InventoryService) GetInventoryItems(inventoryId string) ([]InventoryItemRow, error) {
	return i.getRepo().GetInventoryItems(inventoryId)
}

// GetItemDetails returns full details for a single item (on row selection).
func (i *InventoryService) GetItemDetails(inventoryId, itemType, itemKey string) (*ItemDetails, error) {
	return i.getRepo().GetItemDetails(inventoryId, itemType, itemKey)
}

// RenameInventory updates the inventory name.
func (i *InventoryService) RenameInventory(id, newName string) error {
	return i.getRepo().RenameInventory(id, newName)
}

// DeleteInventory removes an inventory and its items (cascade).
func (i *InventoryService) DeleteInventory(id string) error {
	return i.getRepo().DeleteInventory(id)
}

// ServiceShutdown deletes temporary inventories. Called by Wails when the app exits.
func (i *InventoryService) ServiceShutdown() error {
	return i.getRepo().DeleteTemporaryInventories()
}

func toRepoInventoryItem(in semanticquery.InventoryItem) repos.InventoryItem {
	return repos.InventoryItem{
		Key:           in.Key,
		Type:          in.Type,
		FilePath:      in.FilePath,
		LineStart:     in.LineStart,
		LineEnd:       in.LineEnd,
		RawText:       in.RawText,
		PotentialRefs: in.PotentialRefs,
		Attributes:    in.Attributes,
	}
}

func toQueryInventoryItem(in repos.InventoryItem) semanticquery.InventoryItem {
	return semanticquery.InventoryItem{
		Key:           in.Key,
		Type:          in.Type,
		FilePath:      in.FilePath,
		LineStart:     in.LineStart,
		LineEnd:       in.LineEnd,
		RawText:       in.RawText,
		PotentialRefs: in.PotentialRefs,
	}
}

func toRepoRefs(in []semanticquery.InventoryReference) []repos.ObjectReference {
	out := make([]repos.ObjectReference, 0, len(in))
	for _, r := range in {
		out = append(out, repos.ObjectReference{
			Key:       r.Key,
			Type:      r.Type,
			FilePath:  r.FilePath,
			LineStart: r.LineStart,
			LineEnd:   r.LineEnd,
		})
	}
	return out
}
