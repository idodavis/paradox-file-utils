# session

The live state of one open workspace. Answers *"what does the workspace look like
right now?"* as the user edits — buffers, parse results, and the `model.Index`.

Imports `model` (and through it `parser`/`loc`/`game`). Consumed by `lsp` and
`graph`, which take a `*session.Session` and never build state themselves.

## Responsibilities

- **Buffers** — open file text + `*parser.Result`; edits index unsaved content.
- **One reindex path** — `DidSave` and the file watcher both funnel through a
  single `reindexFile`, which patches only the changed file's defs/refs/edges.
  Re-indexing identical bytes is a no-op, so an editor save followed by the
  watcher seeing the same write does not double-index.
- **Override resolution** — `Resolve(key)` defers entirely to `model.Winner`;
  the session never re-implements FIOS/LIOS.
- **Pool** — `EnsureSession` builds each workspace's session once even under
  concurrent callers (`singleflight`), and emits a ready/reindexed event so the
  frontend refreshes without polling.
- **Watcher** — `fsnotify` on mod roots catches *external* changes (git
  checkout/pull, patcher/merge writes, another editor) and runs the same reindex.

## Files

| File         | Responsibility |
| ------------ | -------------- |
| `doc.go`     | package doc comment (godoc) |
| `session.go` | buffers, DidOpen/Change/Close/Save, reindex, resolve |
| `path.go`    | CanonPath / SamePath / RelPath for index identity |
| `pool.go`    | workspaceID -> *Session, coalesced build, event emit |
| `watcher.go` | fsnotify -> coalesced reindex for external changes |
| `ignore.go`  | `# pmt:ignore` / `# pmt:ignore-next-line` directives |
