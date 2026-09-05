---
title: Workspace Health
description: FIOS/LIOS contests, overlays, depends, dangling refs, and loc coverage.
weight: 5
---

# Workspace Health

Compatibility and localization for **this** workspace.

{{< shot caption="KPI cards + table + load-order chips." >}}

## Compatibility cards

| Card | Meaning |
|------|---------|
| **Contests** | Two origins define the same key. Winner follows [FIOS / LIOS]({{< relref "/concepts/load-order" >}}). |
| **Overlays** | A mod file covers a game file at the same relative path. |
| **Depends** | A mod refers to a key that lives in another attached mod. |
| **Dangling** | A ref with no definition in the live session (typo, missing dependency, or a game patch that removed the key). |

Row detail shows a colorized snippet around the hit.

## Localization cards

For the language in the heading select:

- **Missing** — script wants a key the loc files do not have.
- **Orphans** — loc key with no script ref we know.
- **Untranslated** — English (or source) exists; this language does not.

Coverage lists **every language the mods ship**. The workspace default loc language is an IDE/hover setting.

## Load-order preview

Drag the load-order chips to preview winners **without** writing the workspace. **Save as workspace order** persists the same `SortOrder` as Settings.

After a game patch: pin/rescan the install, then use dangling + overlays here. See [roadmap]({{< relref "/roadmap" >}}).
