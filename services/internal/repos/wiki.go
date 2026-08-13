// Package repos provides database repositories for the application.
package repos

import "github.com/jmoiron/sqlx"

// WikiPatch is a cached wiki patch entry.
type WikiPatch struct {
	GameID      string `json:"gameId" db:"game_id"`
	Version     string `json:"version" db:"version"`
	FetchedAt   string `json:"fetchedAt" db:"fetched_at"`
	SourceURL   string `json:"sourceUrl" db:"source_url"`
	HTMLContent string `json:"htmlContent" db:"html_content"`
}

// WikiRepository handles wiki_patches database operations.
type WikiRepository struct {
	db *sqlx.DB
}

// NewWikiRepository creates a new WikiRepository.
func NewWikiRepository(db *sqlx.DB) *WikiRepository {
	return &WikiRepository{db: db}
}

// Get retrieves a cached patch by game and version.
func (r *WikiRepository) Get(gameID, version string) (*WikiPatch, error) {
	var patch WikiPatch
	err := r.db.Get(&patch,
		`SELECT game_id, version, fetched_at, source_url, html_content FROM wiki_patches WHERE game_id = ? AND version = ?`,
		gameID, version,
	)
	if err != nil {
		return nil, err
	}
	return &patch, nil
}

// Upsert inserts or updates a wiki patch cache entry.
func (r *WikiRepository) Upsert(patch *WikiPatch) error {
	_, err := r.db.Exec(
		`INSERT INTO wiki_patches (game_id, version, fetched_at, source_url, html_content) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(game_id, version) DO UPDATE SET fetched_at=excluded.fetched_at, source_url=excluded.source_url, html_content=excluded.html_content`,
		patch.GameID, patch.Version, patch.FetchedAt, patch.SourceURL, patch.HTMLContent,
	)
	return err
}

// ListByGame returns all cached patches for a game.
func (r *WikiRepository) ListByGame(gameID string) ([]WikiPatch, error) {
	var out []WikiPatch
	err := r.db.Select(&out,
		`SELECT game_id, version, fetched_at, source_url, html_content FROM wiki_patches WHERE game_id = ? ORDER BY version DESC`,
		gameID,
	)
	return out, err
}
