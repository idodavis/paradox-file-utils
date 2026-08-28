// Package lsp answers one editor request against a live session. Each file is
// one LSP feature; handlers use session maps plus the current parse result and
// never re-walk a file to discover references. Completions are never hidden —
// scopes.go only ranks. This package must not import graph.
package lsp
