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
- Local develop/build flows use Taskfile (`task dev`, `task build`) with the Wails v3 CLI (`wails3@v3.0.0-beta.14`).
- Workspace IDE embeds monaco-vscode-api workbench (Monaco + VS Code services); product chrome stays Nuxt.
- **Thin language engine:** no `go:embed` of game knowledge. Install facts come from `model/scan.go` → user-data `Cache` (keyed by game + version). Workspace facts from one-pass `model/index.go` → persisted `Index`. Game-specific extract overrides live only in `game/rules.go` + `game/registry.go`. Wrong or missing `formatVersion` discards the file (no migrations).
- Layered packages with one-way deps: `parser` → `loc` / `game` (leaves) → `model` → `session` → `lsp` / `graph` (siblings; never import each other). There is no `lang/` package and no `manager.go` (`session/pool.go` owns `EnsureSession`).
- The Paradox/Jomini script parser lives under `services/internal/parser/` (UTF-8 byte offsets everywhere in Go). Localization YAML under `services/internal/loc/` (`loc.Property` STRICT/BROAD for required-loc). The only UTF-16 seam is `languageClient.ts` at the Monaco boundary.
- Product direction is workspace-centric: library of workspaces (mods + game versions, grouped by game), workspace IDE, mod patcher, event graph, conflict monitor (FIOS/LIOS via `GetOverrides`), loc coverage page, merge tools, and ad-hoc tools. Patch Center was removed. Event-graph layout is frontend `@dagrejs/dagre` + Vue Flow; Go returns nodes/edges with no coordinates.
- Workspaces support first-launch/retriggerable setup wizard, dropdown switching, user-specified mod paths, default merge output location, multiple game install paths per game, and clear "broken path" marking when folders are missing.
- Semantic models are built from game installs (not mods); app services use the parser alone or parser + semantic model depending on the feature. Merge/patcher use `parser` + `game.KeyIdentity` only (via `MergeService`).
- Wails registers six services: `SettingsService` (settings, version, updater, ResetData), `FileService`, `WorkspaceService`, `LanguageModelService` (session, LSP, health, semantics rebuild, `GetEventGraph`/`GetEventDetail`/`GetOverrides`/`GetLocCoverage`/`LookupLoc`/`GetDependencies`), `PatcherService`, `MergeService`. `DbService` and `LogService` remain internal (SQLite + file logging at startup).
- Frontend uses generated Wails TypeScript bindings via alias `@services` → `frontend/bindings/paradox-modding-tools/services`; models from `@services/internal/repos/models`, `@services/internal/model/models`, and `@services/internal/graph/models`; service functions from `@services/{servicename}`.
- App chrome themes: PMT, pastel, retro, dracula, luxury, business. IDE syntax tokens come from `@codingame/monaco-vscode-theme-defaults-default-extension` (dark_plus / light_plus); all `@codingame/monaco-vscode-*` packages pin the same version via npm overrides.
- Monaco language providers use `registerLanguageClient(() => workspaceId)` — no `useWorkspaceStore()` inside provider callbacks.
- Tool pages use `useWorkspaceStore().ensureReady()` — success means a **live** session (`GetModelStatus().live`), not merely `defCount > 0`.
- On Windows, the parser is pure Go (`CGO_ENABLED=0`). Linux/darwin stay `CGO_ENABLED=1` for Wails WebKit. `go test -race ./services/internal/parser/...` should work on Windows.
