# game

The small, hardcoded **engine grammar** and per-title **identity** for the
supported games (CK3, Vic3, EU5). Answers *"what does this game call things, and
where do they live?"*

Leaf package (no engine dependencies). Used by `model` (extract rules, key
identity), `session`/`lsp`/`graph` (call kinds, FIOS, scope roots), and the
workspace/wiki services (detection, wiki endpoints).

## The data/grammar line (why this stays install-only)

This package holds **only** engine mechanics and static identity — a few small
tables. It bundles **no** game data: objects, vocabulary, structures, and loc are
all derived from the install by `model`. Concretely, `game` holds:

- extract overrides (folder -> definition kind + read mode)
- call kinds (`scripted_effect`/`scripted_trigger`/`scripted_modifier`)
- FIOS kinds (first-in-order-selection; everything else is last-in)
- `KeyIdentity` (EU5 `MODE:` prefix stripping to a canonical key)
- default root scope, and (later) scope-link grammar
- identity: names, script/stage roots, descriptor kind, Steam ids, wiki endpoints

## Files

| File          | Responsibility |
| ------------- | -------------- |
| `doc.go`      | package doc comment (godoc) |
| `registry.go` | `GameInfo` table + `Get`/`All` |
| `detect.go`   | Steam install detection, version read, mod-root recognition |
| `rules.go`    | extract overrides, call/FIOS kinds, `KeyIdentity`, scope roots |

## Ported from

Identity only (not schema data) from toolkit
`packages/server/src/games/{ck3,vic3,eu5}/meta.ts`; detection is PMT-specific.
