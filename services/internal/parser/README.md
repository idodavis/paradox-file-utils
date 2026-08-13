# Paradox script parser

Shared Clausewitz/Jomini **AST only** — no game-specific keyword lists.

## Four layers

1. **AST** (`parser` + `walk`) — structure of `.txt` / `.gui`
2. **Bootstrap** (`semantics/bootstrap/*.json`) — thin per-game roots + path stubs
3. **Install cache** (`semantics/scanner`) — generated semantics from a game install
4. **Workspace index** — SQLite objects / loc keys / edges for a workspace

Localization YAML is a **sibling** package: `services/internal/loc` (not this parser).
