# model

Builds and persists the semantic model. Answers *"what objects/refs/edges exist
in this game + workspace?"* This is where **all** game knowledge is derived from
the install — the package bundles nothing.

Imports the leaf packages `parser`, `loc`, `game`. Consumed by `session` (which
holds a live model) and, through it, `lsp`/`graph`.

## Two artifacts

- **Cache** — the *vanilla* semantic model for one `(game, version)`: definitions,
  the English loc map, field-doc prose, per-kind structure keys, script
  vocabulary, GUI types/props, and metadata keys. Built by `scan.go`, keyed by
  game version, reused across every workspace on that version.
- **Index** — the *workspace* model (mod defs, references, edges, loc map). Built
  by `index.go`.

## Scan sources (all from the install)

| Source | Yields |
| ------ | ------ |
| vanilla `.txt` corpus | defs, per-kind structure keys, script vocabulary, objects |
| `localization/**/*.yml` | English loc map |
| shipped docs (`_*.info`, `*.md`) | field-doc prose, documented structure keys, root scopes |
| `.gui` files | GUI types + props |
| `.metadata/metadata.json` | metadata keys |
| script_docs dump (optional) | effect/trigger/modifier classification + prose enrichment |

The corpus path is the always-available baseline; script_docs only *enriches*
classification and prose when a dump is present.

## Contracts

- **Breaking changes discard, never migrate.** `LoadCache`/`LoadIndex` reject a
  file whose `formatVersion` does not match the current constant; the caller
  rescans.
- Membership only: structure/vocab sets record *that* a key is valid, not how
  often it occurs (no frequency ranking data is persisted).

## Files

| File         | Responsibility |
| ------------ | -------------- |
| `doc.go`     | package doc comment (godoc) |
| `types.go`   | `Def`, `Cache`, `Index`, format-version constants |
| `scan.go`    | install -> `Cache` (defs, vocab, docs, structures, gui, meta) |
| `index.go`   | workspace -> `Index` (mod defs, refs, edges, loc map) |
| `override.go`| FIOS/LIOS override winner |
| `persist.go` | save/load Cache & Index; reject wrong format version |

## Ported from

Toolkit `packages/server/src/index/extract.ts` (definition extraction) and
`scripts/build-structures-json.ts` (doc + corpus harvest), reduced to membership
sets and moved from build-time bundling to runtime scan.
