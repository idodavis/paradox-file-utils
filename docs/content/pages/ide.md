---
title: Workspace IDE
description: monaco-vscode workbench with PMT-owned roots, read-only game files, and Guide.
weight: 3
---

# Workspace IDE

The IDE is a monaco-vscode-api workbench (Monaco + VS Code services) inside PMT chrome. Themes and scale live in the PMT header.

{{< shot caption="IDE: explorer + script editor + Guide open." >}}

## Folders

Roots come from **this workspace**: your mods, then the game files. PMT owns the folder set.

- **Game files** (CK3, Vic3, or EU5) are **read-only**. Edit in a mod folder.
- Search and Ctrl+P index every root.
- Deleting a **mod’s root folder** in the explorer also detaches it from this workspace. Files inside a mod delete normally.

## Tabs

Open editor tabs persist across workspace switch and app close unless you turn that off in [Workspace Settings]({{< relref "/pages/settings" >}}).

## Language features

On Paradox script, loc, and descriptors in **mod** folders:

| Feature | Notes |
|---------|--------|
| Hover | Kind, origin, field info / wiki-backed one-liners, loc text |
| Complete / signature | Session vocab + dump signatures when `script_docs` is present |
| Go to definition / references | Across attached mods + vanilla |
| Symbols / rename | Workspace-aware |
| Format / folding | Jomini CST |
| Diagnostics | Parse errors, loc BOM/header, missing loc, unknown events, missing descriptor |
| Code actions | Including suppressions |

Game files stay read-only, so diagnostics run on mods.

### `# pmt:ignore`

In a script comment:

```
# pmt:ignore
# pmt:ignore-next-line
# pmt:ignore missing-loc
```

That suppresses diagnostics on the same line or the next line. Optional codes limit what is ignored.

## Guide

The **Guide** pull-tab on the editor’s right edge opens a resizable **wiki** pane (search, contents, related pages). It stays closed until you open it.

This is the community wiki, not `script_docs` and not install `_*.info`. See [three kinds of docs]({{< relref "/concepts/docs" >}}).

Snooze or turn off the Guide toast under [Display]({{< relref "/pages/display" >}}).

{{< shot caption="Hover card on a def (kind, docs, origin)." >}}

## Event Graph from hover

Hover an event id (the definition or a `trigger_event`) for **Open in Event Graph**.
