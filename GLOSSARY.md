# Terminology

How to use this file: names here win over older code and comments. Rename in the
owning cleanup phase; do not invent a synonym. Go JSON tags match Vue. Wails
params stay `workspaceID`; Vue locals may be `workspaceId`.

## Guide, not Helper

The product pane beside the IDE editor is **Guide**. The SFC is `GuidePane.vue`.
The monaco-vscode custom view id is `pmt.guide`. Prefs are `pmt.guide.*` /
`ui.guide.*`. RPC is `WikiService.Guide`. Do not say Helper, HelperPane,
`pmt.helper`, `WikiGuide`, or `ide.helperWidth`.

## Three things people call “docs”

These are different pipelines. Never collapse them into one word.

| What | Canonical | Where | What it is |
|---|---|---|---|
| Community wiki | wiki, Guide | `package wiki` | Sanitized MediaWiki HTML. Guide is wiki-only. |
| Console dumps | script_docs | OS user-data `logs/` (CK3) or `docs/` (Vic3/EU5) | In-game `script_docs` output. Silent ingest. Health is `scriptDocsEffects` (indexed effect count), not “folder exists.” |
| Install prose | game info | Shipped `_*.info`, readmes, `*.md` **inside the game install** | Harvested into `fieldInfo` / `fieldInfoByKind`. Never call this script_docs or wiki. Never name the cache field `GameInfo` (collides with `game.GameInfo`). |

Hover one-liners are `session.FieldDoc`: kind `fieldInfo` → global `fieldInfo` → dump `TokenDoc`. Dump signatures stay in `TokenUsage`.

## Install, user data, workshop

| Tree | What |
|---|---|
| **Install** | Steam/Paradox game folder. Vanilla script, GUI, loc, game info. `VanillaCache` is keyed by install id + version. |
| **User data** | `game.UserDataDir`: first existing of launcher `gameDataPath`; Win/mac `~/Documents/Paradox Interactive/<DocsFolderName>`; Linux `$XDG_DATA_HOME` or `~/.local/share/Paradox Interactive/<DocsFolderName>`; Proton `compatdata/<SteamAppID>/pfx/drive_c/users/steamuser/Documents/Paradox Interactive/<DocsFolderName>` (also Flatpak Steam). script_docs and the default `mod/` parent live here. |
| **Workshop / mods** | User-specified mod roots on the workspace. Harvest is RAM-only (`BuildIndex`). |

## Script

Clausewitz/Jomini text (`.txt`, `.gui`). One CST in `parser/jomini`. Loc YAML is
`parser/loc`. No third GUI parser. `MatchExtract` is **def kind**. `ClassifyRel`
is **file bucket** (`script` / `gui` / `loc` / `info` / `meta`).

## Modding practice

Load-order is **LIOS** (last in, only saved / last listed wins). CK3/Vic3 path
overwrite; EU5 also `INJECT:` / `REPLACE:` entry modes. Patch = retarget a mod
onto a new game version.

## Canonical names

| Concept | Canonical | Kill |
|---|---|---|
| Product pane | Guide, `GuidePane`, `pmt.guide`, `ui.guide.*` | Helper, HelperPane, `pmt.helper`, `ui.helper`, `ide.helperWidth`, `WikiGuide` |
| Guide RPC | `WikiService.Guide` | `SessionService.WikiGuide`; query key `wiki-guide` → `guide` |
| Events | `wiki:first`, `wiki:updated` | `docs:wiki-*` |
| Package / load | `package wiki`, `LoadGuides`, `guidesPath()` | `package docs`, `LoadDocs`, persist `docsPath()` |
| script_docs health | `scriptDocsEffects` (int) | `docsPresent`, UI “dumps” |
| Install prose maps | `fieldInfo` / `fieldInfoByKind` | `fieldDocs*`; never `VanillaCache.GameInfo` |
| One hover sentence | `session.FieldDoc` | `FieldDocsForKind`, `Guide.FieldDocs` |
| Dump signature | `TokenUsage` | prose in TokenUsage |
| Dump effect text | `TokenDoc` | stuffing into FieldInfo |
| Origin id on the wire | `"vanilla"` or mod uuid | `""` meaning vanilla |
| Origin display | `originName` | showing the uuid |
| Same origin field | JSON `origin` | `IdeRoot.originId` |
| Def kind | `Def.Kind`, JSON `kind` | `Def.Type` / JSON `type` |
| File bucket | `ClassifyRel` → `script`/`gui`/`loc`/`info`/`meta` | `FileKind`; bucket `"docs"` |
| Absolute path | `path` | `LocEntry.File`, override `file` when absolute |
| Display relative | `rel` | using `path` as the label; `DisplayRel` **fills** `rel` |
| Live session | `LanguageHealth.indexReady` | `ModelStatus.live` |
| Query gate file | `useLiveEnabled.ts` | `useSessionQuery.ts` |
| Load-order rule | LIOS | **LISO** |
| Graph layout | dagre | AGENTS elkjs (do not add elkjs) |

**Keep as different words:** `lsp.Position.character` vs `HoverResult.col`.
`EventFieldInfo` vs cache `fieldInfo`. `Ref.Kind` vs `Def.Kind`. Disk
`DocsFolderName` / `ScriptDocsSubdir`.

## Code map

| Name | Lives in |
|---|---|
| `GameInfo`, `UserDataDir`, `WriteNewMod`, `MatchExtract` | `services/internal/game` |
| `VanillaCache`, `Scan`, `BuildIndex`, `ClassifyRel`, dump parsers | `services/internal/catalog` |
| `FieldDoc`, session query API | `services/internal/session` |
| Wiki persist / sanitize / Guide payload | `services/internal/wiki` |
| Health `scriptDocsEffects` | `SessionService.GetLanguageHealth` |
| Guide UI | `GuidePane.vue` + monaco aux bar (`pmt.guide`) |
| Graph layout | frontend `useGraphLayout` + `@dagrejs/dagre` |
