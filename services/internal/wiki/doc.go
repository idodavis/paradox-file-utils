// Package wiki fetches, sanitizes, and looks up Paradox wiki guides and patch
// notes. It persists game-keyed sidecars under catalog.CacheDir (not VanillaCache).
// WikiService owns Wails RPCs; this package must not import session, lsp, or views.
package wiki
