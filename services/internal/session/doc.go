// Package session holds the live state of one open workspace: file buffers,
// their parse results, VanillaCache, and RAM harvest maps rebuilt on open
// (catalog.BuildIndex). It is the only type that holds both. lsp and views
// call Session methods; public Cache() getters do not exist. There is no
// persisted mod Index.
//
// Files: session.go, pool.go, watcher.go, path.go.
package session
