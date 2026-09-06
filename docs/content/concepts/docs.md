---
title: Three kinds of “docs”
description: Wiki/Guide vs script_docs dumps vs install _*.info — three pipelines.
weight: 4
---

# Three kinds of “docs”

People say “docs” for three different things. PMT keeps them separate on purpose.

| What | In PMT | Where it comes from | What it is |
|------|--------|---------------------|------------|
| Community wiki | **Guide** | Cached MediaWiki HTML | The pane beside the IDE. Search, contents, related pages. |
| Console dumps | `script_docs` | User-data `logs/` (CK3) or `docs/` (Vic3/EU5) | Optional overlay. In-game `script_docs` / `dump_data_types`. Health’s effect chip is **voted** install uses, not “folder exists.” `DumpHint` appears only when dump signatures (`TokenDoc`) are missing. |
| Install prose | game info | `_*.info`, readmes, `*.md` **inside the game install** | Harvested into field info for hover. |

## Guide (wiki)

The [IDE]({{< relref "/pages/ide" >}}) **Guide** pull-tab is the community wiki. [Patch Center]({{< relref "/pages/patch-center" >}}) is the same wiki pipeline, scoped to patch-note pages.

Wiki is orientation. Vanilla files and `script_docs` are the API when you write script.

## script_docs

From the game’s debug console (`~` with `-debug_mode`):

- CK3: writes under Documents `Crusader Kings III/logs/`
- Vic3 / EU5: Documents `<Game>/docs/` (Vic3 also `DumpDataTypes`; EU5 `dump_data_types` under `logs/data_types`)

Dumps are optional. Hover and complete work from the install walk (voted effect/trigger names). Dumps add signatures, unused tokens, and Vic3 dynamic modifier types. If Health shows the optional hint, run `script_docs` in-game once, then Rescan.

## Install `_*.info`

Paradox ships short field descriptions next to script in the install. PMT harvests them for hover one-liners (`fieldInfo`).

Hover in the IDE prefers: kind-specific field info → global field info → dump `TokenDoc`. Signatures stay on the dump side.

## Why this split exists

Collapsing “docs” into one word caused the wrong pane to open and the wrong cache to update. If you file a bug, say **Guide**, **script_docs**, or **game info** — not “docs are missing.”
