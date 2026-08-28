// Package session holds the live state of one open workspace: file buffers, their
// parse results, and the model.Index built from the mods on disk. It answers
// "what does the workspace look like right now?" as the user edits. It owns the
// single reindex path shared by editor saves and the external-change watcher, and
// resolves override winners through model.Winner (never its own logic).
//
// Files: session.go (buffers + DidOpen/Change/Close/Save + reindex + resolve),
// pool.go (workspaceID -> *Session, coalesced construction), watcher.go (fsnotify
// -> reindex for external changes), ignore.go (pmt:ignore comment directives).
package session
