# lsp

Answers one editor request against a live `*session.Session`. One file per LSP
feature; handlers never re-walk a file for references — they use the session
index plus the current `*parser.Result`.

`lsp` must not import `graph`. Completions are never hidden; `scopes.go` only
ranks. Wiki hover uses the map the session already holds (loaded by the Wails
shell).

## Files

| File             | Feature |
| ---------------- | ------- |
| `types.go`       | DTOs + shared position/word helpers |
| `diagnostics.go` | CST/loc errors, missing loc, BOM, descriptor |
| `hover.go`       | def / kind-scoped fieldDocs / wiki |
| `complete.go`    | vocab + defs + descriptor/meta keys |
| `definition.go`  | overlay + vanilla on-demand parse |
| `references.go`  | index refs + defs |
| `rename.go`      | workspace edit for one key |
| `format.go`      | script tab-indent + loc dialect |
| `folding.go`     | block fold ranges |
| `tokens.go`      | semantic highlight spans |
| `signature.go`   | field-doc signature |
| `codeactions.go` | create loc key (UTF-8 BOM), add BOM |
| `symbols.go`     | document + workspace symbols |
| `scopes.go`      | completion ranking only |
