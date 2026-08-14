## Learned User Preferences

- Prefer the smallest clear implementation: cut duplication, dead-end helpers, and over-engineering; do not add code for backwards compatibility.
- Change only what was explicitly requested; avoid unrelated refactors or drive-by edits.
- Keep plans short, actionable, and free of stale wording after iterative edits; ask clarifying questions when requirements are ambiguous.
- Prefer game-agnostic shared parser/semantics logic; keep game-specific facts and outputs under each game's own files/dirs.
- Prefer a JSON semantic model when it can hold the needed baseline structure; avoid SQLite unless relationships or volume truly require it.
- Put performance-sensitive work in Go behind the Wails 3 bridge when that stays maintainable; keep UI logic in the frontend otherwise.
- Prefer "patch" (not "update"/"campaign") for mod version retargeting, and prefer clear product names over ambiguous jargon.
- Prefer Nuxt UI component patterns (`:items` and similar) over ad-hoc `v-if`/`v-model` control flow where Nuxt UI fits.
- Prefer denser layouts where editor/diff/merge content dominates the viewport and path/config chrome stays compact.
- When cleaning code, keep useful comments and user-facing behavior; reduce complexity without expanding scope.

## Learned Workspace Facts

- Paradox Modding Tools is a Wails v3 desktop app (Go backend, Vue 3 + Nuxt UI + Pinia frontend) for Paradox modders, licensed under GPL-3.0-or-later.
- CK3, EU5 (partial), and Vic3 are the current game targets; the longer-term goal is broader Paradox-title support.
- Local develop/build flows use Taskfile (`task dev`, `task build`) with the Wails v3 CLI (`wails3`).
- Workspace IDE embeds monaco-vscode-api workbench (Monaco + VS Code services); product chrome stays Nuxt.
- The Paradox/Jomini script parser lives under `services/internal/parser/`; classification under `services/internal/semantics/` (bootstrap JSON + install scan cache); localization YAML under `services/internal/loc/`.
- Per-game thin bootstrap packs (`semantics/bootstrap/{ck3,eu5,vic3}.json`) are prefilled from installs; large semantics are generated at scan-time via `semantics/scanner`.
- Product direction is workspace-centric: library of workspaces (mods + game versions, grouped by game), workspace IDE, patch center, mod patcher, event graph, and ad-hoc tools.
- Workspaces support first-launch/retriggerable setup wizard, dropdown switching, user-specified mod paths, default merge output location, multiple game install paths per game, and clear "broken path" marking when folders are missing.
- Semantic models are intended to be built from game installs (not mods); app services use the parser alone or parser + semantic model depending on the feature.
- Frontend uses generated Wails TypeScript bindings via alias `@services` → `frontend/bindings/paradox-modding-tools/services`; models from `@services/internal/repos/models`, service functions from `@services/{servicename}`.
