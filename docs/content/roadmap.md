---
title: Roadmap
description: Loose direction — Health-first patching, GUI preview, CK3 map editor, IDE undock.
weight: 7
---

# Roadmap

This is **direction**, not a calendar. Order can change. File issues against a [roadmap bucket](https://github.com/idodavis/paradox-modding-tools/issues/new/choose) if you want to push on one of these.

## Automatic game-update patching

After a game patch today:

1. Pin or keep `latest`, rescan the install.
2. Reopen the workspace.
3. Read [Workspace Health]({{< relref "/pages/health" >}}) — dangling refs and game-file overlays.

Later: suggested **file-level retarget in the IDE** (open the overlay, copy forward what you still own).

## GUI preview

A future **views** feature using the same Jomini CST already used for `.gui`. Goal is a **structural** HTML/CSS widget tree (`hbox` / `vbox` / `button`) so you can see nesting — not a pixel-perfect launcher clone.

## Map / title editor

**CK3 political layer first.**

- Read `provinces.png` (color-index) + `definition.csv` + `landed_titles` tree.
- Go builds a raster overlay; Vue is the canvas + inspector.
- Write `zz_pmt_map*.txt` plus loc (UTF-8 BOM) into a **workspace mod**.

The game’s `-mapeditor` still owns heightmap bake, locators, and splines. Vic3 `state_regions` and EU5 locations are later adapters on the same idea.

## Multi-window

Wails v3 can open extra windows. Today `main.go` creates one.

{{< mermaid >}}
flowchart LR
  subgraph now [Now]
    oneWin[One webview]
    oneSess[One live session]
  end
  subgraph later [Later]
    ideWin[IDE window]
    toolWin[Tool pages]
    pool[Pool: N sessions or share one]
  end
  oneWin --> ideWin
  oneSess --> pool
  pool --> toolWin
{{< /mermaid >}}

- **Same workspace, extra windows:** feasible (hash router + Pinia). **IDE undock** is the useful first step — each webview gets its own monaco-vscode workbench talking to the **same** Go session.
- **Two workspaces at once:** needs `EnsureSession` to keep more than one session live.
- Undocking Health / Graph / Publish is the same extra-window work once the pool rule is decided.

Target: **IDE undock first; full-app windows after the one-session rule is relaxed.**

## Game coverage

Deeper EU5 extract rules and Guide/wiki coverage. More loc languages in hover without a full rescan (sidecar already works this way).
