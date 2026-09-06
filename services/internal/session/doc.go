// Package session holds the live state of one open workspace: file buffers,
// their parse results, VanillaCache, and RAM harvest maps rebuilt on open
// (catalog.BuildIndex). It is the only type that holds both. lsp and views
// call Session methods; public Cache() getters do not exist. There is no
// persisted mod Index.
//
// Files: session.go holds the mutable half — the Session type, lifecycle, and
// the buffer/reindex machinery. query.go is the read-only API that lsp and
// views are limited to. scope.go walks the declared scope types, inspect.go
// fills the hover card, cursor.go resolves an offset. Then pool.go, watcher.go,
// path.go.
package session
